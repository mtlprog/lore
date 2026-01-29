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
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"golang.org/x/sync/semaphore"
)

const (
	horizonPageLimit = 200
	concurrentLimit  = 10
)

// fetchAllAssetHolders returns all account IDs holding the specified asset.
func (s *Syncer) fetchAllAssetHolders(ctx context.Context, code, issuer string) ([]string, error) {
	var accountIDs []string
	cursor := ""

	for {
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
// Returns SyncResult with stats and failed account list.
// Returns error if failure rate exceeds the configured threshold.
func (s *Syncer) syncAccounts(ctx context.Context, accountIDs []string) (*SyncResult, error) {
	sem := semaphore.NewWeighted(concurrentLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedAccounts []string

	totalCount := len(accountIDs)

	for _, id := range accountIDs {
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, fmt.Errorf("acquire semaphore: %w", err)
		}

		wg.Add(1)
		go func(accountID string) {
			defer wg.Done()
			defer sem.Release(1)

			if err := s.syncSingleAccount(ctx, accountID); err != nil {
				s.logger.Error("failed to sync account", "account_id", accountID, "error", err)
				mu.Lock()
				failedAccounts = append(failedAccounts, accountID)
				mu.Unlock()
				return
			}
		}(id)
	}

	wg.Wait()

	failedCount := len(failedAccounts)
	failureRate := float64(0)
	if totalCount > 0 {
		failureRate = float64(failedCount) / float64(totalCount)
	}

	result := &SyncResult{
		FailedAccounts:  failedAccounts,
		AccountFailRate: failureRate,
	}

	if failedCount > 0 {
		s.logger.Error("accounts failed to sync",
			"failed_count", failedCount,
			"total_count", totalCount,
			"failed_accounts", failedAccounts[:min(10, failedCount)],
		)

		if failureRate > s.failureThreshold {
			return result, fmt.Errorf("sync failed: %d/%d accounts failed (%.1f%%), threshold %.1f%%",
				failedCount, totalCount, failureRate*100, s.failureThreshold*100)
		}
	}

	return result, nil
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

	// Parse balances using lo.Map
	data.Balances = lo.Map(acc.Balances, func(bal horizon.Balance, _ int) Balance {
		if bal.Type == "native" {
			return Balance{
				AssetCode:   "XLM",
				AssetIssuer: "",
				Balance:     decimal.RequireFromString(bal.Balance),
			}
		}
		return Balance{
			AssetCode:   bal.Code,
			AssetIssuer: bal.Issuer,
			Balance:     decimal.RequireFromString(bal.Balance),
		}
	})

	// Parse ManageData
	data.Metadata, data.Relationships, data.DelegateTo, data.CouncilDelegateTo, data.CouncilReady = parseManageData(acc.Data)

	// Extract primary name from metadata (key="Name", index="")
	for _, m := range data.Metadata {
		if m.Key == "Name" && m.Index == "" {
			data.Name = m.Value
			break
		}
	}

	return data
}

// relationTypeStrings maps string to RelationType for parsing.
var relationTypeStrings = map[string]RelationType{
	"MyPart":            RelationMyPart,
	"PartOf":            RelationPartOf,
	"RecommendToMTLA":   RelationRecommendToMTLA,
	"OneFamily":         RelationOneFamily,
	"Spouse":            RelationSpouse,
	"Guardian":          RelationGuardian,
	"Ward":              RelationWard,
	"Sympathy":          RelationSympathy,
	"Love":              RelationLove,
	"Divorce":           RelationDivorce,
	"A":                 RelationA,
	"B":                 RelationB,
	"C":                 RelationC,
	"D":                 RelationD,
	"Employer":          RelationEmployer,
	"Employee":          RelationEmployee,
	"Contractor":        RelationContractor,
	"Client":            RelationClient,
	"Partnership":       RelationPartnership,
	"Collaboration":     RelationCollaboration,
	"OwnershipFull":     RelationOwnershipFull,
	"OwnershipMajority": RelationOwnershipMajority,
	"OwnershipMinority": RelationOwnershipMinority,
	"Owner":             RelationOwner,
	"OwnerMajority":     RelationOwnerMajority,
	"OwnerMinority":     RelationOwnerMinority,
	"WelcomeGuest":      RelationWelcomeGuest,
	"FactionMember":     RelationFactionMember,
}

// relationTypePrefixes for parsing relationship keys (ordered by length for proper matching).
var relationTypePrefixes = []string{
	"RecommendToMTLA",
	"OwnershipMajority", "OwnershipMinority", "OwnershipFull",
	"OwnerMajority", "OwnerMinority",
	"FactionMember",
	"Collaboration", "Partnership",
	"WelcomeGuest",
	"Contractor",
	"OneFamily",
	"Employer", "Employee",
	"Guardian",
	"Sympathy",
	"Divorce",
	"Client",
	"Spouse",
	"MyPart", "PartOf",
	"Owner",
	"Love",
	"Ward",
	"A", "B", "C", "D",
}

// parseManageData extracts metadata, relationships, delegate_to, council_delegate_to, and council_ready from account data.
func parseManageData(rawData map[string]string) ([]Metadata, []Relationship, *string, *string, bool) {
	var metadata []Metadata
	var relationships []Relationship
	var delegateTo *string
	var councilDelegateTo *string
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
		case "mtla_c_delegate":
			// mtla_c_delegate = "ready" marks as council-ready
			// mtla_c_delegate = account ID means council delegation to that account
			if strings.ToLower(value) == "ready" {
				councilReady = true
			} else if len(value) == 56 && strings.HasPrefix(value, "G") {
				councilDelegateTo = &value
			}
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
			return items[i].index < items[j].index
		})

		for _, item := range items {
			metadata = append(metadata, Metadata{
				Key:   baseKey,
				Index: item.index,
				Value: item.value,
			})
		}
	}

	return metadata, relationships, delegateTo, councilDelegateTo, councilReady
}

// parseRelationship attempts to parse a key/value pair as a relationship.
// Key format: "Type" or "Type0", "Type1", "Type002" (with optional index suffix)
// Value format: Account ID (56 chars starting with G)
// RelationIndex preserves the original suffix string (e.g., "002" vs "2") to avoid conflicts.
func parseRelationship(key, value string) *Relationship {
	// Value must be a valid Stellar account ID
	if len(value) != 56 || !strings.HasPrefix(value, "G") {
		return nil
	}

	// Try each known relation type as prefix
	for _, prefix := range relationTypePrefixes {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		rest := key[len(prefix):]

		// Verify rest is numeric only (if present)
		if len(rest) > 0 {
			if _, err := strconv.Atoi(rest); err != nil {
				continue
			}
		}

		relType, ok := relationTypeStrings[prefix]
		if !ok {
			continue
		}

		return &Relationship{
			TargetAccountID: value,
			RelationType:    relType,
			RelationIndex:   rest, // Keep original suffix string (empty string for no suffix)
		}
	}

	return nil
}

// parseNumberedKey extracts the base key and index suffix from keys like "Website0", "Name".
// Returns the original suffix string to preserve leading zeros (e.g., "002" vs "2").
func parseNumberedKey(key string) (baseKey string, index string) {
	re := regexp.MustCompile(`^(.+?)(\d*)$`)
	matches := re.FindStringSubmatch(key)
	if matches == nil {
		return key, ""
	}

	return matches[1], matches[2]
}

// decodeBase64 decodes a base64-encoded string, returning empty string on error.
// Logs a warning on decode failure as this may indicate corrupted data.
func decodeBase64(s string) string {
	if s == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		slog.Warn("failed to decode base64 - data may be corrupted", "error", err, "input_length", len(s))
		return ""
	}
	return strings.TrimSpace(string(decoded))
}

// findBalance finds a specific balance from the account balances using lo.Find.
func findBalance(balances []Balance, code, issuer string) decimal.Decimal {
	bal, found := lo.Find(balances, func(b Balance) bool {
		return b.AssetCode == code && b.AssetIssuer == issuer
	})
	if !found {
		return decimal.Zero
	}
	return bal.Balance
}

// getMTLAPBalance returns the MTLAP balance from account data.
func getMTLAPBalance(data *AccountData) decimal.Decimal {
	return findBalance(data.Balances, config.TokenMTLAP, config.TokenIssuer)
}

// getMTLACBalance returns the MTLAC balance from account data.
func getMTLACBalance(data *AccountData) decimal.Decimal {
	return findBalance(data.Balances, config.TokenMTLAC, config.TokenIssuer)
}

// getNativeBalance returns the XLM balance from account data.
func getNativeBalance(data *AccountData) decimal.Decimal {
	return findBalance(data.Balances, "XLM", "")
}
