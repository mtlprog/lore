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
	Trustlines    []Trustline
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
