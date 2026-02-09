package api

// PaginatedResponse wraps a list of items with pagination metadata.
type PaginatedResponse struct {
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination holds offset-based pagination metadata.
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// AccountListItem represents an account in list responses.
type AccountListItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"` // "person", "corporate", "synthetic"
	MTLAPBalance    float64 `json:"mtlap_balance"`
	MTLACBalance    float64 `json:"mtlac_balance"`
	MTLAXBalance    float64 `json:"mtlax_balance"`
	TotalXLMValue   float64 `json:"total_xlm_value"`
	ReputationScore float64 `json:"reputation_score,omitempty"`
	ReputationGrade string  `json:"reputation_grade,omitempty"`
	IsCouncilReady  bool    `json:"is_council_ready,omitempty"`
	ReceivedVotes   int     `json:"received_votes,omitempty"`
}

// AccountDetailResponse represents full account detail.
type AccountDetailResponse struct {
	ID            string                         `json:"id"`
	Name          string                         `json:"name"`
	About         string                         `json:"about,omitempty"`
	Websites      []string                       `json:"websites,omitempty"`
	Tags          []string                       `json:"tags,omitempty"`
	IsCorporate   bool                           `json:"is_corporate"`
	TotalXLMValue float64                        `json:"total_xlm_value"`
	Trustlines    []TrustlineResponse            `json:"trustlines,omitempty"`
	LPShares      []LPShareResponse              `json:"lp_shares,omitempty"`
	TrustRating   *TrustRatingResponse           `json:"trust_rating,omitempty"`
	Reputation    *ReputationResponse            `json:"reputation,omitempty"`
	Categories    []RelationshipCategoryResponse `json:"categories,omitempty"`
}

// TrustlineResponse represents a single asset trustline.
type TrustlineResponse struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	Balance     string `json:"balance"`
	Limit       string `json:"limit,omitempty"`
}

// LPShareResponse represents a liquidity pool share.
type LPShareResponse struct {
	PoolID       string          `json:"pool_id"`
	ShareBalance string          `json:"share_balance"`
	SharePercent string          `json:"share_percent"`
	ReserveA     ReserveResponse `json:"reserve_a"`
	ReserveB     ReserveResponse `json:"reserve_b"`
	XLMValue     float64         `json:"xlm_value"`
}

// ReserveResponse represents a liquidity pool reserve.
type ReserveResponse struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	Amount      string `json:"amount"`
}

// TrustRatingResponse represents aggregated trust ratings.
type TrustRatingResponse struct {
	CountA int     `json:"count_a"`
	CountB int     `json:"count_b"`
	CountC int     `json:"count_c"`
	CountD int     `json:"count_d"`
	Total  int     `json:"total"`
	Score  float64 `json:"score"`
	Grade  string  `json:"grade"`
}

// ReputationResponse represents a weighted reputation score.
type ReputationResponse struct {
	WeightedScore float64 `json:"weighted_score"`
	BaseScore     float64 `json:"base_score"`
	Grade         string  `json:"grade"`
	RatingCountA  int     `json:"rating_count_a"`
	RatingCountB  int     `json:"rating_count_b"`
	RatingCountC  int     `json:"rating_count_c"`
	RatingCountD  int     `json:"rating_count_d"`
	TotalRatings  int     `json:"total_ratings"`
	TotalWeight   float64 `json:"total_weight"`
}

// ReputationGraphResponse represents a 2-level reputation graph.
type ReputationGraphResponse struct {
	TargetAccountID string                   `json:"target_account_id"`
	TargetName      string                   `json:"target_name"`
	Score           *ReputationResponse      `json:"score,omitempty"`
	Level1Nodes     []ReputationNodeResponse `json:"level1_nodes"`
	Level2Nodes     []ReputationNodeResponse `json:"level2_nodes"`
}

// ReputationNodeResponse represents a node in the reputation graph.
type ReputationNodeResponse struct {
	AccountID    string  `json:"account_id"`
	Name         string  `json:"name"`
	Rating       string  `json:"rating"`
	Weight       float64 `json:"weight"`
	PortfolioXLM float64 `json:"portfolio_xlm"`
	Connections  int     `json:"connections"`
	OwnScore     float64 `json:"own_score"`
	Distance     int     `json:"distance"`
}

// RelationshipCategoryResponse groups relationships by category.
type RelationshipCategoryResponse struct {
	Name          string                 `json:"name"`
	Color         string                 `json:"color"`
	Relationships []RelationshipResponse `json:"relationships"`
}

// RelationshipResponse represents a single relationship.
type RelationshipResponse struct {
	Type        string `json:"type"`
	TargetID    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	Direction   string `json:"direction"`
	IsMutual    bool   `json:"is_mutual"`
	IsConfirmed bool   `json:"is_confirmed"`
}

// StatsResponse represents aggregate statistics.
type StatsResponse struct {
	TotalAccounts  int     `json:"total_accounts"`
	TotalPersons   int     `json:"total_persons"`
	TotalCompanies int     `json:"total_companies"`
	TotalSynthetic int     `json:"total_synthetic"`
	TotalXLMValue  float64 `json:"total_xlm_value"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
