package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mtlprog/lore/internal/model"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
)

// ErrTokenNotFound is returned when a token does not exist.
var ErrTokenNotFound = errors.New("token not found")

// cacheEntry holds a cached value with expiration time.
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// cache is a simple TTL-based in-memory cache.
type cache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	ttl   time.Duration
}

// newCache creates a new cache with the specified TTL.
func newCache(ttl time.Duration) *cache {
	return &cache{
		items: make(map[string]cacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a value from the cache. Expired entries are cleaned up.
func (c *cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Clean up expired entry
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.data, true
}

// Set stores a value in the cache.
func (c *cache) Set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// StellarService provides access to Stellar network data.
type StellarService struct {
	client     *horizonclient.Client
	httpClient *http.Client // Shared HTTP client for external requests
	nftCache   *cache       // Caches NFT metadata (1 hour TTL)
	tomlCache  *cache       // Caches stellar.toml content (1 hour TTL)
}

// NewStellarService creates a new Stellar service with the given Horizon URL.
func NewStellarService(horizonURL string) *StellarService {
	return &StellarService{
		client: &horizonclient.Client{HorizonURL: horizonURL},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		nftCache:  newCache(1 * time.Hour),
		tomlCache: newCache(1 * time.Hour),
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

// GetRawAccountData fetches raw ManageData from Horizon for init form parsing.
func (s *StellarService) GetRawAccountData(ctx context.Context, accountID string) (map[string]string, error) {
	acc, err := s.client.AccountDetail(horizonclient.AccountRequest{AccountID: accountID})
	if err != nil {
		return nil, err
	}
	return acc.Data, nil
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
	tags := parseTagKeys(acc.Data)

	trustlines := make([]model.Trustline, 0, len(acc.Balances))
	for _, bal := range acc.Balances {
		// Skip LP shares - they are displayed separately
		if bal.Type == "liquidity_pool_shares" {
			continue
		}
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
		Tags:       tags,
		Trustlines: trustlines,
	}, nil
}

// GetAccountSequence returns the current sequence number for an account.
func (s *StellarService) GetAccountSequence(ctx context.Context, accountID string) (int64, error) {
	acc, err := s.client.AccountDetail(horizonclient.AccountRequest{AccountID: accountID})
	if err != nil {
		return 0, err
	}
	return acc.Sequence, nil
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

// parseTagKeys extracts tag names from "Tag*" keys (e.g., "TagBelgrade" -> "Belgrade").
// The value of each tag key is an account ID which is ignored for display purposes.
func parseTagKeys(data map[string]string) []string {
	var tags []string
	for key := range data {
		if strings.HasPrefix(key, "Tag") {
			if len(key) <= 3 {
				slog.Debug("skipping tag key with no suffix", "key", key)
				continue
			}
			tagName := key[3:] // Strip "Tag" prefix
			tags = append(tags, tagName)
		}
	}
	sort.Strings(tags)
	return tags
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

// GetAccountOperations returns operations for an account (cursor-based pagination).
// Filters out spam operations (claimable balance, small XLM payments < 1).
// Fetches additional pages if needed to fill the requested limit.
func (s *StellarService) GetAccountOperations(ctx context.Context, accountID, cursor string, limit int) (*model.OperationsPage, error) {
	const (
		fetchSize     = 50 // Fetch more than needed to allow for filtering
		maxIterations = 5  // Max API calls to prevent hanging on spam-heavy accounts
	)

	var filteredOps []model.Operation
	currentCursor := cursor
	iterations := 0
	hasMoreInBlockchain := true
	lastCursor := ""

	for len(filteredOps) < limit && hasMoreInBlockchain && iterations < maxIterations {
		iterations++

		req := horizonclient.OperationRequest{
			ForAccount: accountID,
			Limit:      uint(fetchSize),
			Order:      horizonclient.OrderDesc, // Newest first
		}
		if currentCursor != "" {
			req.Cursor = currentCursor
		}

		page, err := s.client.Operations(req)
		if err != nil {
			return nil, err
		}

		hasMoreInBlockchain = len(page.Embedded.Records) == fetchSize

		for _, op := range page.Embedded.Records {
			currentCursor = op.PagingToken()

			converted := convertOperation(op)

			// Filter spam operations
			if isSpamOperation(converted) {
				continue
			}

			filteredOps = append(filteredOps, converted)
			lastCursor = currentCursor

			// Stop if we have enough
			if len(filteredOps) >= limit {
				break
			}
		}
	}

	// Determine if there's more data
	hasMore := len(filteredOps) > limit || (len(filteredOps) == limit && hasMoreInBlockchain)

	// Trim to requested limit
	if len(filteredOps) > limit {
		filteredOps = filteredOps[:limit]
		lastCursor = filteredOps[limit-1].ID
	}

	return &model.OperationsPage{
		Operations: filteredOps,
		NextCursor: lastCursor,
		HasMore:    hasMore,
	}, nil
}

// isSpamOperation returns true if the operation should be filtered out.
func isSpamOperation(op model.Operation) bool {
	// Filter claimable balance operations (spam)
	if op.Type == "create_claimable_balance" || op.Type == "claim_claimable_balance" {
		return true
	}

	// Filter small XLM payments (< 1 XLM) - common spam pattern
	if op.Type == "payment" && op.AssetCode == "XLM" {
		amount, err := strconv.ParseFloat(op.Amount, 64)
		if err == nil && amount < 1 {
			return true
		}
	}

	return false
}

// GetTransactionDetail returns a transaction with its operations.
func (s *StellarService) GetTransactionDetail(ctx context.Context, txHash string) (*model.Transaction, error) {
	tx, err := s.client.TransactionDetail(txHash)
	if err != nil {
		return nil, err
	}

	// Fetch operations for this transaction
	opsReq := horizonclient.OperationRequest{
		ForTransaction: txHash,
		Limit:          200, // Max operations per transaction is 100, so 200 is safe
	}
	opsPage, err := s.client.Operations(opsReq)
	if err != nil {
		return nil, err
	}

	ops := make([]model.Operation, 0, len(opsPage.Embedded.Records))
	for _, op := range opsPage.Embedded.Records {
		ops = append(ops, convertOperation(op))
	}

	return &model.Transaction{
		Hash:           tx.Hash,
		Successful:     tx.Successful,
		Ledger:         int(tx.Ledger),
		CreatedAt:      tx.LedgerCloseTime.Format("2006-01-02 15:04:05"),
		SourceAccount:  tx.Account,
		FeeCharged:     formatStroops(tx.FeeCharged),
		MemoType:       tx.MemoType,
		Memo:           tx.Memo,
		OperationCount: int(tx.OperationCount),
		Operations:     ops,
	}, nil
}

// formatStroops converts stroops (int64) to XLM string.
func formatStroops(stroops int64) string {
	xlm := float64(stroops) / 10000000
	return fmt.Sprintf("%.7f", xlm)
}

// convertOperation converts a Horizon operation to a model.Operation.
func convertOperation(op operations.Operation) model.Operation {
	base := op.GetBase()

	result := model.Operation{
		ID:              base.ID,
		Type:            base.Type,
		TypeDisplay:     operationTypeDisplay(base.Type),
		TypeCategory:    operationTypeCategory(base.Type),
		CreatedAt:       base.LedgerCloseTime.Format("2006-01-02 15:04:05"),
		TransactionHash: base.TransactionHash,
		SourceAccount:   base.SourceAccount,
	}

	// Type-specific field extraction
	switch typed := op.(type) {
	case operations.Payment:
		result.From = typed.From
		result.To = typed.To
		result.Amount = typed.Amount
		result.AssetCode = assetCodeDisplay(typed.Asset.Type, typed.Asset.Code)
		result.AssetIssuer = typed.Asset.Issuer

	case operations.CreateAccount:
		result.From = typed.Funder
		result.To = typed.Account
		result.StartingBalance = typed.StartingBalance

	case operations.ChangeTrust:
		result.AssetCode = assetCodeDisplay(typed.Asset.Type, typed.Asset.Code)
		result.AssetIssuer = typed.Asset.Issuer
		result.TrustLimit = typed.Limit

	case operations.ManageData:
		result.DataName = typed.Name
		result.DataValue = decodeBase64(typed.Value)

	case operations.PathPaymentStrictSend:
		result.From = typed.From
		result.To = typed.To
		result.SourceAmount = typed.SourceAmount
		result.SourceAsset = assetCodeDisplay(typed.SourceAssetType, typed.SourceAssetCode)
		result.DestAmount = typed.Amount
		result.DestAsset = assetCodeDisplay(typed.Asset.Type, typed.Asset.Code)

	case operations.PathPayment:
		result.From = typed.From
		result.To = typed.To
		result.SourceAmount = typed.SourceAmount
		result.SourceAsset = assetCodeDisplay(typed.SourceAssetType, typed.SourceAssetCode)
		result.DestAmount = typed.Amount
		result.DestAsset = assetCodeDisplay(typed.Asset.Type, typed.Asset.Code)

	case operations.ManageSellOffer:
		result.Selling = assetCodeDisplay(typed.SellingAssetType, typed.SellingAssetCode)
		result.Buying = assetCodeDisplay(typed.BuyingAssetType, typed.BuyingAssetCode)
		result.Amount = typed.Amount
		result.Price = typed.Price
		result.OfferID = fmt.Sprintf("%d", typed.OfferID)

	case operations.ManageBuyOffer:
		result.Selling = assetCodeDisplay(typed.SellingAssetType, typed.SellingAssetCode)
		result.Buying = assetCodeDisplay(typed.BuyingAssetType, typed.BuyingAssetCode)
		result.Amount = typed.Amount
		result.Price = typed.Price
		result.OfferID = fmt.Sprintf("%d", typed.OfferID)

	case operations.CreatePassiveSellOffer:
		result.Selling = assetCodeDisplay(typed.SellingAssetType, typed.SellingAssetCode)
		result.Buying = assetCodeDisplay(typed.BuyingAssetType, typed.BuyingAssetCode)
		result.Amount = typed.Amount
		result.Price = typed.Price

	case operations.AccountMerge:
		result.From = typed.Account
		result.To = typed.Into

	case operations.LiquidityPoolDeposit:
		result.Amount = typed.SharesReceived

	case operations.LiquidityPoolWithdraw:
		result.Amount = typed.Shares
	}

	return result
}

// operationTypeDisplay returns human-readable operation type.
func operationTypeDisplay(opType string) string {
	displays := map[string]string{
		"create_account":                   "Create Account",
		"payment":                          "Payment",
		"path_payment_strict_receive":      "Path Payment",
		"path_payment_strict_send":         "Path Payment",
		"manage_sell_offer":                "Sell Offer",
		"manage_buy_offer":                 "Buy Offer",
		"create_passive_sell_offer":        "Passive Offer",
		"set_options":                      "Set Options",
		"change_trust":                     "Change Trust",
		"allow_trust":                      "Allow Trust",
		"account_merge":                    "Account Merge",
		"inflation":                        "Inflation",
		"manage_data":                      "Manage Data",
		"bump_sequence":                    "Bump Sequence",
		"create_claimable_balance":         "Create Claimable",
		"claim_claimable_balance":          "Claim Balance",
		"begin_sponsoring_future_reserves": "Begin Sponsor",
		"end_sponsoring_future_reserves":   "End Sponsor",
		"revoke_sponsorship":               "Revoke Sponsor",
		"clawback":                         "Clawback",
		"clawback_claimable_balance":       "Clawback Claim",
		"set_trust_line_flags":             "Trust Flags",
		"liquidity_pool_deposit":           "LP Deposit",
		"liquidity_pool_withdraw":          "LP Withdraw",
		"invoke_host_function":             "Contract Call",
		"extend_footprint_ttl":             "Extend TTL",
		"restore_footprint":                "Restore",
	}
	if d, ok := displays[opType]; ok {
		return d
	}
	return opType
}

// operationTypeCategory returns the category for color-coding.
func operationTypeCategory(opType string) string {
	switch opType {
	case "payment", "path_payment_strict_receive", "path_payment_strict_send":
		return "payment"
	case "change_trust", "allow_trust", "set_trust_line_flags":
		return "trust"
	case "manage_data":
		return "data"
	case "manage_sell_offer", "manage_buy_offer", "create_passive_sell_offer",
		"liquidity_pool_deposit", "liquidity_pool_withdraw":
		return "dex"
	case "create_account", "account_merge", "set_options", "bump_sequence":
		return "account"
	default:
		return "other"
	}
}

// assetCodeDisplay returns display code for an asset (XLM for native).
func assetCodeDisplay(assetType, assetCode string) string {
	if assetType == "native" {
		return "XLM"
	}
	return assetCode
}

// GetTokenDetail returns information about a token from the Horizon /assets endpoint.
func (s *StellarService) GetTokenDetail(ctx context.Context, code, issuer string) (*model.TokenDetail, error) {
	req := horizonclient.AssetRequest{
		ForAssetCode:   code,
		ForAssetIssuer: issuer,
		Limit:          1,
	}

	page, err := s.client.Assets(req)
	if err != nil {
		return nil, err
	}

	if len(page.Embedded.Records) == 0 {
		return nil, ErrTokenNotFound
	}

	asset := page.Embedded.Records[0]

	// Get issuer account details for name and home_domain
	issuerAcc, err := s.client.AccountDetail(horizonclient.AccountRequest{AccountID: issuer})
	var issuerName, homeDomain string
	if err != nil {
		if IsNotFound(err) {
			slog.Debug("issuer account not found", "issuer", issuer)
		} else {
			slog.Warn("failed to fetch issuer account details", "issuer", issuer, "error", err)
		}
	} else {
		issuerName = decodeBase64(issuerAcc.Data["Name"])
		homeDomain = issuerAcc.HomeDomain
	}
	if issuerName == "" {
		issuerName = issuer[:6] + "..." + issuer[len(issuer)-6:]
	}

	return &model.TokenDetail{
		AssetCode:   asset.Code,
		AssetIssuer: asset.Issuer,
		IssuerName:  issuerName,
		NumAccounts: int64(asset.Accounts.Authorized),
		Amount:      asset.Balances.Authorized,
		HomeDomain:  homeDomain,
	}, nil
}

// GetTokenOrderbook returns the orderbook for a token against XLM.
func (s *StellarService) GetTokenOrderbook(ctx context.Context, code, issuer string, limit int) (*model.TokenOrderbook, error) {
	req := horizonclient.OrderBookRequest{
		SellingAssetType:   horizonclient.AssetType4,
		SellingAssetCode:   code,
		SellingAssetIssuer: issuer,
		BuyingAssetType:    horizonclient.AssetTypeNative,
		Limit:              uint(limit),
	}

	// Adjust asset type based on code length
	if len(code) > 4 {
		req.SellingAssetType = horizonclient.AssetType12
	}

	orderbook, err := s.client.OrderBook(req)
	if err != nil {
		return nil, err
	}

	bids := make([]model.OrderbookEntry, 0, len(orderbook.Bids))
	for _, bid := range orderbook.Bids {
		bids = append(bids, model.OrderbookEntry{
			Price:  bid.Price,
			Amount: bid.Amount,
		})
	}

	asks := make([]model.OrderbookEntry, 0, len(orderbook.Asks))
	for _, ask := range orderbook.Asks {
		asks = append(asks, model.OrderbookEntry{
			Price:  ask.Price,
			Amount: ask.Amount,
		})
	}

	return &model.TokenOrderbook{
		Bids: bids,
		Asks: asks,
	}, nil
}

// ipfsGateway is the IPFS gateway URL for fetching NFT metadata.
const ipfsGateway = "https://ipfs.io/ipfs/"

// ipfsMetadata represents the JSON structure of SEP-0039 NFT metadata.
type ipfsMetadata struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	FullDescription string `json:"fulldescription"`
	Image           string `json:"image"`
	File            string `json:"file"`
	ContentType     string `json:"content_type"`
}

// GetIssuerNFTMetadata checks for ipfshash or ipfshash-{CODE} in issuer's ManageData.
// Results are cached for 1 hour to avoid repeated IPFS requests.
func (s *StellarService) GetIssuerNFTMetadata(ctx context.Context, issuerID, assetCode string) (*model.NFTMetadata, error) {
	cacheKey := issuerID + "/" + assetCode

	// Check cache first
	if cached, ok := s.nftCache.Get(cacheKey); ok {
		if cached == nil {
			return nil, nil // Cached negative result
		}
		if meta, ok := cached.(*model.NFTMetadata); ok {
			return meta, nil
		}
		slog.Error("unexpected type in NFT cache", "key", cacheKey, "type", fmt.Sprintf("%T", cached))
		// Treat as cache miss and continue with fresh fetch
	}

	acc, err := s.client.AccountDetail(horizonclient.AccountRequest{AccountID: issuerID})
	if err != nil {
		return nil, err
	}

	// Check for ipfshash-{CODE} first, then generic ipfshash
	var ipfsHash string
	if hash := decodeBase64(acc.Data["ipfshash-"+assetCode]); hash != "" {
		ipfsHash = hash
	} else if hash := decodeBase64(acc.Data["ipfshash"]); hash != "" {
		ipfsHash = hash
	}

	if ipfsHash == "" {
		s.nftCache.Set(cacheKey, nil) // Cache negative result
		return nil, nil               // Not an NFT
	}

	// Validate IPFS hash format (CIDv0: base58, CIDv1: base32/base36)
	// IPFS CIDs only contain alphanumeric characters
	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(ipfsHash) {
		slog.Warn("invalid IPFS hash format", "issuer", issuerID, "hash", ipfsHash)
		s.nftCache.Set(cacheKey, nil)
		return nil, nil
	}

	// Fetch metadata from IPFS
	resp, err := s.httpClient.Get(ipfsGateway + ipfsHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IPFS metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPFS returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read IPFS response: %w", err)
	}

	var meta ipfsMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse IPFS metadata: %w", err)
	}

	// Decode base64 fulldescription if present
	fullDesc := meta.FullDescription
	if decoded, err := base64.StdEncoding.DecodeString(meta.FullDescription); err == nil {
		fullDesc = string(decoded)
	}

	// Convert IPFS URLs to gateway URLs
	imageURL := meta.Image
	if strings.HasPrefix(imageURL, "ipfs://") {
		imageURL = ipfsGateway + strings.TrimPrefix(imageURL, "ipfs://")
	}

	fileURL := meta.File
	if strings.HasPrefix(fileURL, "ipfs://") {
		fileURL = ipfsGateway + strings.TrimPrefix(fileURL, "ipfs://")
	}

	result := &model.NFTMetadata{
		Name:            meta.Name,
		Description:     meta.Description,
		FullDescription: fullDesc,
		ImageURL:        imageURL,
		FileURL:         fileURL,
		ContentType:     meta.ContentType,
	}

	s.nftCache.Set(cacheKey, result)
	return result, nil
}

// stellarTomlCurrency represents a currency entry in stellar.toml for TOML parsing.
type stellarTomlCurrency struct {
	Code        string `toml:"code"`
	Issuer      string `toml:"issuer"`
	Name        string `toml:"name"`
	Description string `toml:"desc"`
	Image       string `toml:"image"`
}

// stellarToml represents the structure of a stellar.toml file.
type stellarToml struct {
	Currencies []stellarTomlCurrency `toml:"CURRENCIES"`
}

// tomlCacheEntry stores both the parsed currency and raw content.
type tomlCacheEntry struct {
	currency *model.StellarTomlCurrency
	content  string
}

// FetchStellarToml fetches and parses the stellar.toml file for an issuer.
// Results are cached for 1 hour to avoid repeated HTTP requests.
func (s *StellarService) FetchStellarToml(ctx context.Context, homeDomain string) (*model.StellarTomlCurrency, string, error) {
	if homeDomain == "" {
		return nil, "", nil
	}

	// Validate homeDomain to prevent SSRF attacks
	if !isValidHomeDomain(homeDomain) {
		slog.Warn("invalid home domain rejected", "domain", homeDomain)
		return nil, "", nil
	}

	// Check cache first
	if cached, ok := s.tomlCache.Get(homeDomain); ok {
		if entry, ok := cached.(*tomlCacheEntry); ok {
			return entry.currency, entry.content, nil
		}
		slog.Error("unexpected type in TOML cache", "domain", homeDomain, "type", fmt.Sprintf("%T", cached))
		// Treat as cache miss and continue with fresh fetch
	}

	url := "https://" + homeDomain + "/.well-known/stellar.toml"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch stellar.toml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("stellar.toml returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, "", fmt.Errorf("failed to read stellar.toml: %w", err)
	}

	var tomlData stellarToml
	if _, err := toml.Decode(string(body), &tomlData); err != nil {
		return nil, "", fmt.Errorf("failed to parse stellar.toml: %w", err)
	}

	// Return first currency (we'll match by code/issuer in the handler)
	var currency *model.StellarTomlCurrency
	if len(tomlData.Currencies) > 0 {
		c := tomlData.Currencies[0]
		currency = &model.StellarTomlCurrency{
			Code:        c.Code,
			Issuer:      c.Issuer,
			Name:        c.Name,
			Description: c.Description,
			Image:       c.Image,
		}
	}

	// Cache the result
	s.tomlCache.Set(homeDomain, &tomlCacheEntry{
		currency: currency,
		content:  string(body),
	})

	return currency, string(body), nil
}

// FindCurrencyInToml finds a currency by code and issuer in stellar.toml content.
func FindCurrencyInToml(tomlContent, code, issuer string) *model.StellarTomlCurrency {
	var tomlData stellarToml
	if _, err := toml.Decode(tomlContent, &tomlData); err != nil {
		slog.Error("failed to re-parse stellar.toml content", "error", err)
		return nil
	}

	for _, c := range tomlData.Currencies {
		if c.Code == code && c.Issuer == issuer {
			return &model.StellarTomlCurrency{
				Code:        c.Code,
				Issuer:      c.Issuer,
				Name:        c.Name,
				Description: c.Description,
				Image:       c.Image,
			}
		}
	}
	return nil
}

// isValidHomeDomain validates that a home domain is safe to fetch.
// Prevents SSRF attacks by rejecting localhost, internal IPs, and malformed domains.
func isValidHomeDomain(domain string) bool {
	// Reject domains with path traversal or port numbers
	if strings.Contains(domain, "/") || strings.Contains(domain, "..") || strings.Contains(domain, ":") {
		return false
	}

	// Reject localhost and common internal hostnames
	lower := strings.ToLower(domain)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") ||
		lower == "internal" || strings.HasSuffix(lower, ".internal") ||
		strings.HasSuffix(lower, ".local") {
		return false
	}

	// Reject IP addresses (both IPv4 and IPv6)
	if addr, err := netip.ParseAddr(domain); err == nil {
		// Valid IP address - reject private/loopback ranges
		if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
			return false
		}
		// Also reject any direct IP addresses for safety
		return false
	}

	// Must contain at least one dot (valid domain)
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}
