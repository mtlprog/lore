package service

import (
	"context"
	"encoding/base64"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mtlprog/lore/internal/model"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
)

// StellarService provides access to Stellar network data.
type StellarService struct {
	client *horizonclient.Client
}

// NewStellarService creates a new Stellar service with the given Horizon URL.
func NewStellarService(horizonURL string) *StellarService {
	return &StellarService{
		client: &horizonclient.Client{HorizonURL: horizonURL},
	}
}

// GetAccountsWithAsset returns accounts holding the specified asset.
func (s *StellarService) GetAccountsWithAsset(ctx context.Context, code, issuer, cursor string, limit int) (*model.AccountsPage, error) {
	req := horizonclient.AccountsRequest{
		Asset: code + ":" + issuer,
		Limit: uint(limit),
		Order: horizonclient.OrderAsc,
	}
	if cursor != "" {
		req.Cursor = cursor
	}

	page, err := s.client.Accounts(req)
	if err != nil {
		return nil, err
	}

	accounts := make([]model.AccountSummary, 0, len(page.Embedded.Records))
	var nextCursor string

	for _, acc := range page.Embedded.Records {
		name := decodeBase64(acc.Data["Name"])
		if name == "" {
			name = acc.ID[:6] + "..." + acc.ID[len(acc.ID)-6:]
		}

		balance := findAssetBalance(acc.Balances, code, issuer)

		accounts = append(accounts, model.AccountSummary{
			ID:      acc.ID,
			Name:    name,
			Balance: balance,
		})

		nextCursor = acc.PagingToken()
	}

	return &model.AccountsPage{
		Accounts: accounts,
		Pagination: model.Pagination{
			NextCursor: nextCursor,
			HasMore:    len(page.Embedded.Records) == limit,
		},
	}, nil
}

// GetAccountDetail returns detailed information about an account.
func (s *StellarService) GetAccountDetail(ctx context.Context, accountID string) (*model.AccountDetail, error) {
	acc, err := s.client.AccountDetail(horizonclient.AccountRequest{AccountID: accountID})
	if err != nil {
		return nil, err
	}

	name := decodeBase64(acc.Data["Name"])
	if name == "" {
		name = accountID[:6] + "..." + accountID[len(accountID)-6:]
	}

	about := decodeBase64(acc.Data["About"])
	websites := parseNumberedDataKeys(acc.Data, "Website")

	trustlines := make([]model.Trustline, 0, len(acc.Balances))
	for _, bal := range acc.Balances {
		if bal.Type == "native" {
			trustlines = append(trustlines, model.Trustline{
				AssetCode:   "XLM",
				AssetIssuer: "native",
				Balance:     bal.Balance,
				Limit:       "",
			})
		} else {
			trustlines = append(trustlines, model.Trustline{
				AssetCode:   bal.Code,
				AssetIssuer: bal.Issuer,
				Balance:     bal.Balance,
				Limit:       bal.Limit,
			})
		}
	}

	// Sort trustlines by balance (descending) to show highest-value holdings first.
	// Parse errors are ignored - Horizon API guarantees valid numeric strings.
	sort.Slice(trustlines, func(i, j int) bool {
		balI, _ := strconv.ParseFloat(trustlines[i].Balance, 64)
		balJ, _ := strconv.ParseFloat(trustlines[j].Balance, 64)
		return balI > balJ
	})

	return &model.AccountDetail{
		ID:         acc.ID,
		Name:       name,
		About:      about,
		Websites:   websites,
		Trustlines: trustlines,
	}, nil
}

// findAssetBalance finds the balance for a specific asset.
func findAssetBalance(balances []horizon.Balance, code, issuer string) string {
	for _, bal := range balances {
		if bal.Code == code && bal.Issuer == issuer {
			return bal.Balance
		}
	}
	return "0"
}

// parseNumberedDataKeys extracts values from numbered data keys like "Website", "Website1", "Website0002".
func parseNumberedDataKeys(data map[string]string, prefix string) []string {
	type numbered struct {
		num   int
		value string
	}

	var items []numbered
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `(\d*)$`)

	for key, val := range data {
		matches := re.FindStringSubmatch(key)
		if matches == nil {
			continue
		}

		decoded := decodeBase64(val)
		if decoded == "" {
			continue
		}

		num := 0
		if matches[1] != "" {
			num, _ = strconv.Atoi(matches[1])
		}

		items = append(items, numbered{num: num, value: decoded})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].num < items[j].num
	})

	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.value
	}

	return result
}

// decodeBase64 decodes a base64-encoded string, returning empty string on error.
func decodeBase64(s string) string {
	if s == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		slog.Debug("failed to decode base64", "input", s, "error", err)
		return ""
	}
	return strings.TrimSpace(string(decoded))
}

// IsNotFound checks if the error is a "not found" error from Horizon.
func IsNotFound(err error) bool {
	if hErr, ok := err.(*horizonclient.Error); ok && hErr.Response != nil {
		return hErr.Response.StatusCode == 404
	}
	return false
}
