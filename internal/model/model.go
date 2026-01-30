package model

// AccountSummary represents a token holder in the list view.
type AccountSummary struct {
	ID      string
	Name    string
	Balance string
}

// AccountDetail represents full account information.
type AccountDetail struct {
	ID            string
	Name          string
	About         string
	Websites      []string
	Tags          []string
	Trustlines    []Trustline
	NFTTrustlines []Trustline // Trustlines where balance == "0.0000001" (NFTs)
	Categories    []RelationshipCategory
	TrustRating   *TrustRating // nil if no ratings
	TotalXLMValue float64      // Portfolio value in XLM (for corporate accounts)
	IsCorporate   bool         // true if account holds MTLAC
}

// Trustline represents a single asset trustline.
type Trustline struct {
	AssetCode   string
	AssetIssuer string
	Balance     string
	Limit       string
}

// Pagination holds cursor information for paginated results.
type Pagination struct {
	NextCursor string
	HasMore    bool
}

// AccountsPage represents a paginated list of accounts.
type AccountsPage struct {
	Accounts   []AccountSummary
	Pagination Pagination
}

// Relationship represents a relationship for display.
type Relationship struct {
	Type        string // e.g., "Spouse", "Employer"
	TargetID    string // Full account ID
	TargetName  string // Name or truncated ID
	Direction   string // "outgoing" (→) or "incoming" (←)
	IsMutual    bool   // Same relationship exists in both directions
	IsConfirmed bool   // MyPart/PartOf pair verified
}

// RelationshipCategory groups relationships by category.
type RelationshipCategory struct {
	Name          string // "FAMILY", "WORK", etc.
	Color         string // CSS color
	Relationships []Relationship
	IsEmpty       bool
}

// TrustRating for display.
type TrustRating struct {
	CountA  int
	CountB  int
	CountC  int
	CountD  int
	Total   int
	Score   float64
	Grade   string // "A", "B+", "C", etc.
	Percent int    // For progress bar width
}

// Operation represents a Stellar operation for display.
type Operation struct {
	ID              string
	Type            string // "payment", "create_account", etc.
	TypeDisplay     string // Human-readable: "Payment", "Create Account"
	TypeCategory    string // "payment", "trust", "data", "dex", "account", "other"
	CreatedAt       string // Formatted timestamp
	TransactionHash string
	SourceAccount   string
	// Type-specific fields (flattened for template simplicity)
	Amount          string
	AssetCode       string
	AssetIssuer     string
	From            string
	To              string
	DataName        string
	DataValue       string
	StartingBalance string
	TrustLimit      string
	// Path payment fields
	SourceAmount string
	SourceAsset  string
	DestAmount   string
	DestAsset    string
	// DEX offer fields
	Selling string
	Buying  string
	Price   string
	OfferID string
}

// OperationsPage for cursor-based pagination.
type OperationsPage struct {
	Operations []Operation
	NextCursor string
	HasMore    bool
}

// Transaction for detail page.
type Transaction struct {
	Hash           string
	Successful     bool
	Ledger         int
	CreatedAt      string // Formatted timestamp
	SourceAccount  string
	FeeCharged     string
	MemoType       string
	Memo           string
	OperationCount int
	Operations     []Operation
}

// TokenDetail - information about a token.
type TokenDetail struct {
	AssetCode   string
	AssetIssuer string
	IssuerName  string // Name from account metadata
	NumAccounts int64  // Number of trustlines
	Amount      string // Total supply
	BestBid     string // Best buy price (XLM)
	BestAsk     string // Best sell price (XLM)
	Description string // From stellar.toml
	ImageURL    string // Token image
	HomeDomain  string // Issuer home domain
	IsNFT       bool
	NFTMetadata *NFTMetadata
}

// NFTMetadata - SEP-0039 metadata.
type NFTMetadata struct {
	Name            string
	Description     string // Decoded description
	FullDescription string // base64 decoded fulldescription
	ImageURL        string // ipfs.io gateway URL
	FileURL         string // For non-image files
	ContentType     string
}

// OrderbookEntry - entry from orderbook.
type OrderbookEntry struct {
	Price  string
	Amount string
}

// TokenOrderbook - orderbook summary.
type TokenOrderbook struct {
	Bids []OrderbookEntry
	Asks []OrderbookEntry
}

// StellarTomlCurrency represents a currency entry in stellar.toml.
type StellarTomlCurrency struct {
	Code        string
	Issuer      string
	Name        string
	Description string
	Image       string
}
