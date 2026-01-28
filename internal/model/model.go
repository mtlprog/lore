package model

// AccountSummary represents a token holder in the list view.
type AccountSummary struct {
	ID      string
	Name    string
	Balance string
}

// AccountDetail represents full account information.
type AccountDetail struct {
	ID         string
	Name       string
	About      string
	Websites   []string
	Trustlines []Trustline
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
