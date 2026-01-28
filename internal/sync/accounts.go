package sync

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mtlprog/lore/internal/config"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"golang.org/x/sync/semaphore"
)

const (
	horizonPageLimit = 200
	concurrentLimit  = 10
)

// AccountData holds parsed account information from Horizon.
type AccountData struct {
	ID            string
	Balances      []Balance
	Metadata      []Metadata
	Relationships []Relationship
	DelegateTo    *string
	CouncilReady  bool
}

// Balance represents an account balance.
type Balance struct {
	AssetCode   string
	AssetIssuer string
	Balance     string
}

// Metadata represents account metadata entry.
type Metadata struct {
	Key   string
	Index string
	Value string
}

// Relationship represents a relationship between accounts.
type Relationship struct {
	TargetAccountID string
	RelationType    string
	RelationIndex   string
}

// fetchAllAssetHolders returns all account IDs holding the specified asset.
func (s *Syncer) fetchAllAssetHolders(ctx context.Context, code, issuer string) ([]string, error) {
	var accountIDs []string
	cursor := ""

	for {
		// Check for context cancellation between iterations
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		req := horizonclient.AccountsRequest{
			Asset: code + ":" + issuer,
			Limit: horizonPageLimit,
			Order: horizonclient.OrderAsc,
		}
		if cursor != "" {
			req.Cursor = cursor
		}

		page, err := s.horizon.Accounts(req)
		if err != nil {
			return nil, fmt.Errorf("fetch accounts page: %w", err)
		}

		for _, acc := range page.Embedded.Records {
			accountIDs = append(accountIDs, acc.ID)
			cursor = acc.PagingToken()
		}

		if len(page.Embedded.Records) < horizonPageLimit {
			break
		}
	}

	return accountIDs, nil
}

// syncAccounts fetches and stores account details concurrently.
// Returns error if more than 10% of accounts fail to sync.
func (s *Syncer) syncAccounts(ctx context.Context, accountIDs map[string]struct{}) error {
	sem := semaphore.NewWeighted(concurrentLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedAccounts []string

	totalCount := len(accountIDs)

	for id := range accountIDs {
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("acquire semaphore: %w", err)
		}

		wg.Add(1)
		go func(accountID string) {
			defer wg.Done()
			defer sem.Release(1)

			if err := s.syncSingleAccount(ctx, accountID); err != nil {
				slog.Error("failed to sync account", "account_id", accountID, "error", err)
				mu.Lock()
				failedAccounts = append(failedAccounts, accountID)
				mu.Unlock()
				return
			}
		}(id)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	failedCount := len(failedAccounts)
	if failedCount > 0 {
		slog.Error("accounts failed to sync",
			"failed_count", failedCount,
			"total_count", totalCount,
			"failed_accounts", failedAccounts[:min(10, failedCount)],
		)

		// Return error if failure rate exceeds 10%
		failureRate := float64(failedCount) / float64(totalCount)
		if failureRate > 0.1 {
			return fmt.Errorf("sync failed: %d/%d accounts failed (%.1f%%)", failedCount, totalCount, failureRate*100)
		}
	}

	return nil
}

// syncSingleAccount fetches and stores a single account.
func (s *Syncer) syncSingleAccount(ctx context.Context, accountID string) error {
	acc, err := s.horizon.AccountDetail(horizonclient.AccountRequest{AccountID: accountID})
	if err != nil {
		return fmt.Errorf("fetch account detail: %w", err)
	}

	data := parseAccountData(&acc)

	if err := s.repo.UpsertAccount(ctx, data); err != nil {
		return fmt.Errorf("upsert account: %w", err)
	}

	if err := s.repo.UpsertBalances(ctx, accountID, data.Balances); err != nil {
		return fmt.Errorf("upsert balances: %w", err)
	}

	if err := s.repo.UpsertMetadata(ctx, accountID, data.Metadata); err != nil {
		return fmt.Errorf("upsert metadata: %w", err)
	}

	if err := s.repo.UpsertRelationships(ctx, accountID, data.Relationships); err != nil {
		return fmt.Errorf("upsert relationships: %w", err)
	}

	return nil
}

// parseAccountData extracts structured data from a Horizon account.
func parseAccountData(acc *horizon.Account) *AccountData {
	data := &AccountData{
		ID:           acc.ID,
		CouncilReady: false,
	}

	// Parse balances
	for _, bal := range acc.Balances {
		if bal.Asset.Type == "native" {
			data.Balances = append(data.Balances, Balance{
				AssetCode:   "XLM",
				AssetIssuer: "",
				Balance:     bal.Balance,
			})
		} else {
			data.Balances = append(data.Balances, Balance{
				AssetCode:   bal.Asset.Code,
				AssetIssuer: bal.Asset.Issuer,
				Balance:     bal.Balance,
			})
		}
	}

	// Parse ManageData
	data.Metadata, data.Relationships, data.DelegateTo, data.CouncilReady = parseManageData(acc.Data)

	return data
}

// Known relation type prefixes
var relationTypes = []string{
	"MyPart", "PartOf", "RecommendToMTLA",
	"OneFamily", "Spouse", "Guardian", "Ward", "Sympathy", "Love", "Divorce",
	"A", "B", "C", "D",
	"Employer", "Employee", "Contractor", "Client", "Partnership", "Collaboration",
	"OwnershipFull", "OwnershipMajority", "OwnershipMinority",
	"Owner", "OwnerMajority", "OwnerMinority",
	"WelcomeGuest", "FactionMember",
}

// parseManageData extracts metadata, relationships, delegate_to, and council_ready from account data.
func parseManageData(rawData map[string]string) ([]Metadata, []Relationship, *string, bool) {
	var metadata []Metadata
	var relationships []Relationship
	var delegateTo *string
	councilReady := false

	// Collect numbered keys for grouping
	numberedData := make(map[string][]struct {
		index string
		value string
	})

	for key, rawValue := range rawData {
		value := decodeBase64(rawValue)
		if value == "" {
			continue
		}

		// Check for special keys
		switch key {
		case "mtla_delegate":
			if len(value) == 56 && strings.HasPrefix(value, "G") {
				delegateTo = &value
			}
			continue
		case "mtla_council_ready":
			councilReady = value == "1" || strings.ToLower(value) == "true"
			continue
		}

		// Check for relationship pattern
		if rel := parseRelationship(key, value); rel != nil {
			relationships = append(relationships, *rel)
			continue
		}

		// Parse numbered keys (Name, Name0, Website, Website1, etc.)
		baseKey, index := parseNumberedKey(key)
		numberedData[baseKey] = append(numberedData[baseKey], struct {
			index string
			value string
		}{index: index, value: value})
	}

	// Convert numbered data to metadata, sorted by index
	for baseKey, items := range numberedData {
		sort.Slice(items, func(i, j int) bool {
			ni, _ := strconv.Atoi(items[i].index)
			nj, _ := strconv.Atoi(items[j].index)
			return ni < nj
		})

		for _, item := range items {
			metadata = append(metadata, Metadata{
				Key:   baseKey,
				Index: item.index,
				Value: item.value,
			})
		}
	}

	return metadata, relationships, delegateTo, councilReady
}

// parseRelationship attempts to parse a key as a relationship.
func parseRelationship(key, _ string) *Relationship {
	// Try each known relation type as prefix
	for _, relType := range relationTypes {
		if !strings.HasPrefix(key, relType) {
			continue
		}

		rest := key[len(relType):]
		if len(rest) < 56 {
			continue
		}

		// Extract account ID (first 56 characters after prefix)
		targetID := rest[:56]

		// Verify target looks like a Stellar account ID
		if !strings.HasPrefix(targetID, "G") {
			continue
		}

		// Extract optional index (remaining characters after account ID)
		relIndex := "0"
		if len(rest) > 56 {
			relIndex = rest[56:]
			// Verify index is numeric
			if _, err := strconv.Atoi(relIndex); err != nil {
				continue
			}
		}

		return &Relationship{
			TargetAccountID: targetID,
			RelationType:    relType,
			RelationIndex:   relIndex,
		}
	}

	return nil
}

// parseNumberedKey extracts the base key and index from keys like "Website0", "Name".
func parseNumberedKey(key string) (baseKey string, index string) {
	re := regexp.MustCompile(`^(.+?)(\d*)$`)
	matches := re.FindStringSubmatch(key)
	if matches == nil {
		return key, "0"
	}

	baseKey = matches[1]
	index = matches[2]
	if index == "" {
		index = "0"
	}
	return
}

// decodeBase64 decodes a base64-encoded string, returning empty string on error.
func decodeBase64(s string) string {
	if s == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		slog.Debug("failed to decode base64", "error", err, "input_length", len(s))
		return ""
	}
	return strings.TrimSpace(string(decoded))
}

// findBalance finds a specific balance from the account balances.
func findBalance(balances []Balance, code, issuer string) string {
	for _, bal := range balances {
		if bal.AssetCode == code && bal.AssetIssuer == issuer {
			return bal.Balance
		}
	}
	return "0"
}

// getMTLAPBalance returns the MTLAP balance from account data.
func getMTLAPBalance(data *AccountData) string {
	return findBalance(data.Balances, config.TokenMTLAP, config.TokenIssuer)
}

// getMTLACBalance returns the MTLAC balance from account data.
func getMTLACBalance(data *AccountData) string {
	return findBalance(data.Balances, config.TokenMTLAC, config.TokenIssuer)
}

// getNativeBalance returns the XLM balance from account data.
func getNativeBalance(data *AccountData) string {
	return findBalance(data.Balances, "XLM", "")
}
