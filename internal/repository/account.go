package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/database"
	"github.com/samber/lo"
)

// AccountRepository handles account data access.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates a new account repository.
// Returns error if pool is nil.
func NewAccountRepository(pool *pgxpool.Pool) (*AccountRepository, error) {
	if pool == nil {
		return nil, errors.New("database pool is required")
	}
	return &AccountRepository{pool: pool}, nil
}

// Stats holds aggregate statistics.
type Stats struct {
	TotalAccounts  int
	TotalPersons   int
	TotalCompanies int
	TotalSynthetic int
	TotalXLMValue  float64
}

// PersonRow represents a person (MTLAP holder) from the database.
type PersonRow struct {
	AccountID      string
	Name           string
	MTLAPBalance   float64
	IsCouncilReady bool
	ReceivedVotes  int
}

// CorporateRow represents a corporate account (MTLAC holder) from the database.
type CorporateRow struct {
	AccountID     string
	Name          string
	MTLACBalance  float64
	TotalXLMValue float64
}

// SyntheticRow represents a synthetic account (MTLAX trustline holder) from the database.
type SyntheticRow struct {
	AccountID        string
	Name             string
	MTLAXBalance     float64
	ReputationScore  float64
	ReputationWeight float64
}

// GetStats returns aggregate statistics.
func (r *AccountRepository) GetStats(ctx context.Context) (*Stats, error) {
	query, args, err := database.QB.
		Select(
			"COUNT(*) AS total_accounts",
			"COUNT(*) FILTER (WHERE mtlap_balance > 0 AND mtlap_balance <= 5) AS total_persons",
			"COUNT(*) FILTER (WHERE mtlac_balance > 0 AND mtlac_balance <= 4) AS total_companies",
			"COUNT(*) FILTER (WHERE mtlax_balance IS NOT NULL) AS total_synthetic",
			"COALESCE(SUM(total_xlm_value), 0) AS total_xlm_value",
		).
		From("accounts").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build stats query: %w", err)
	}

	var stats Stats
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&stats.TotalAccounts,
		&stats.TotalPersons,
		&stats.TotalCompanies,
		&stats.TotalSynthetic,
		&stats.TotalXLMValue,
	)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}

	return &stats, nil
}

// GetPersons returns MTLAP holders with their names.
func (r *AccountRepository) GetPersons(ctx context.Context, limit int, offset int) ([]PersonRow, error) {
	query, args, err := database.QB.
		Select(
			"a.account_id",
			"COALESCE(m.data_value, CONCAT(LEFT(a.account_id, 6), '...', RIGHT(a.account_id, 6))) AS name",
			"a.mtlap_balance",
			"a.is_council_ready",
			"a.received_votes",
		).
		From("accounts a").
		LeftJoin("account_metadata m ON a.account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''").
		Where("a.mtlap_balance > 0 AND a.mtlap_balance <= 5").
		OrderBy("a.mtlap_balance DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build persons query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query persons: %w", err)
	}
	defer rows.Close()

	var persons []PersonRow
	for rows.Next() {
		var p PersonRow
		if err := rows.Scan(&p.AccountID, &p.Name, &p.MTLAPBalance, &p.IsCouncilReady, &p.ReceivedVotes); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		persons = append(persons, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persons rows: %w", err)
	}
	return persons, nil
}

// GetCorporate returns MTLAC holders with their names and portfolio values.
func (r *AccountRepository) GetCorporate(ctx context.Context, limit int, offset int) ([]CorporateRow, error) {
	query, args, err := database.QB.
		Select(
			"a.account_id",
			"COALESCE(m.data_value, CONCAT(LEFT(a.account_id, 6), '...', RIGHT(a.account_id, 6))) AS name",
			"a.mtlac_balance",
			"a.total_xlm_value",
		).
		From("accounts a").
		LeftJoin("account_metadata m ON a.account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''").
		Where("a.mtlac_balance > 0 AND a.mtlac_balance <= 4").
		OrderBy("a.mtlac_balance DESC", "a.total_xlm_value DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build corporate query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query corporate: %w", err)
	}
	defer rows.Close()

	var corporate []CorporateRow
	for rows.Next() {
		var c CorporateRow
		if err := rows.Scan(&c.AccountID, &c.Name, &c.MTLACBalance, &c.TotalXLMValue); err != nil {
			return nil, fmt.Errorf("scan corporate: %w", err)
		}
		corporate = append(corporate, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate corporate rows: %w", err)
	}
	return corporate, nil
}

// GetSynthetic returns MTLAX trustline holders sorted by reputation score.
func (r *AccountRepository) GetSynthetic(ctx context.Context, limit int, offset int) ([]SyntheticRow, error) {
	query, args, err := database.QB.
		Select(
			"a.account_id",
			"COALESCE(m.data_value, CONCAT(LEFT(a.account_id, 6), '...', RIGHT(a.account_id, 6))) AS name",
			"COALESCE(a.mtlax_balance, 0) AS mtlax_balance",
			"COALESCE(rs.weighted_score, 0) AS reputation_score",
			"COALESCE(rs.total_weight, 0) AS reputation_weight",
		).
		From("accounts a").
		LeftJoin("account_metadata m ON a.account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''").
		LeftJoin("reputation_scores rs ON a.account_id = rs.account_id").
		Where("a.mtlax_balance IS NOT NULL").
		OrderBy("COALESCE(rs.weighted_score, 0) DESC", "COALESCE(rs.total_weight, 0) DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build synthetic query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query synthetic: %w", err)
	}
	defer rows.Close()

	var synthetic []SyntheticRow
	for rows.Next() {
		var s SyntheticRow
		if err := rows.Scan(&s.AccountID, &s.Name, &s.MTLAXBalance, &s.ReputationScore, &s.ReputationWeight); err != nil {
			return nil, fmt.Errorf("scan synthetic: %w", err)
		}
		synthetic = append(synthetic, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate synthetic rows: %w", err)
	}
	return synthetic, nil
}

// RelationshipRow represents a relationship from the database.
type RelationshipRow struct {
	SourceAccountID string
	TargetAccountID string
	TargetName      string
	RelationType    string
	RelationIndex   string
	Direction       string // "outgoing" or "incoming"
}

// TrustRating represents aggregated trust ratings for an account.
type TrustRating struct {
	CountA int
	CountB int
	CountC int
	CountD int
	Total  int
	Score  float64 // Weighted average (A=4, B=3, C=2, D=1)
}

// GetRelationships returns all relationships for an account (both directions).
func (r *AccountRepository) GetRelationships(ctx context.Context, accountID string) ([]RelationshipRow, error) {
	// Use raw SQL with UNION ALL to get both outgoing and incoming relationships
	// Squirrel doesn't handle UNION well with placeholder renumbering
	query := `
		SELECT
			r.source_account_id,
			r.target_account_id,
			COALESCE(m.data_value, CONCAT(LEFT(r.target_account_id, 6), '...', RIGHT(r.target_account_id, 6))) AS target_name,
			r.relation_type,
			r.relation_index,
			'outgoing' AS direction
		FROM relationships r
		LEFT JOIN account_metadata m ON r.target_account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''
		WHERE r.source_account_id = $1
		  AND r.relation_type NOT IN ('A', 'B', 'C', 'D')
		UNION ALL
		SELECT
			r.source_account_id,
			r.target_account_id,
			COALESCE(m.data_value, CONCAT(LEFT(r.source_account_id, 6), '...', RIGHT(r.source_account_id, 6))) AS target_name,
			r.relation_type,
			r.relation_index,
			'incoming' AS direction
		FROM relationships r
		LEFT JOIN account_metadata m ON r.source_account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''
		WHERE r.target_account_id = $1
		  AND r.relation_type NOT IN ('A', 'B', 'C', 'D')
		ORDER BY relation_type, relation_index
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query relationships: %w", err)
	}
	defer rows.Close()

	var relationships []RelationshipRow
	for rows.Next() {
		var rel RelationshipRow
		if err := rows.Scan(&rel.SourceAccountID, &rel.TargetAccountID, &rel.TargetName, &rel.RelationType, &rel.RelationIndex, &rel.Direction); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}
		relationships = append(relationships, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationship rows: %w", err)
	}
	return relationships, nil
}

// GetTrustRatings returns aggregated A/B/C/D ratings given TO this account.
func (r *AccountRepository) GetTrustRatings(ctx context.Context, accountID string) (*TrustRating, error) {
	query, args, err := database.QB.
		Select(
			"COUNT(*) FILTER (WHERE relation_type = 'A') AS count_a",
			"COUNT(*) FILTER (WHERE relation_type = 'B') AS count_b",
			"COUNT(*) FILTER (WHERE relation_type = 'C') AS count_c",
			"COUNT(*) FILTER (WHERE relation_type = 'D') AS count_d",
		).
		From("relationships").
		Where("target_account_id = ?", accountID).
		Where("relation_type IN ('A', 'B', 'C', 'D')").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build trust ratings query: %w", err)
	}

	var rating TrustRating
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&rating.CountA,
		&rating.CountB,
		&rating.CountC,
		&rating.CountD,
	)
	if err != nil {
		return nil, fmt.Errorf("query trust ratings: %w", err)
	}

	rating.Total = rating.CountA + rating.CountB + rating.CountC + rating.CountD
	if rating.Total > 0 {
		// Weighted score: A=4, B=3, C=2, D=1
		rating.Score = float64(rating.CountA*4+rating.CountB*3+rating.CountC*2+rating.CountD*1) / float64(rating.Total)
	}

	return &rating, nil
}

// GetConfirmedRelationships returns all confirmed relationships for an account.
func (r *AccountRepository) GetConfirmedRelationships(ctx context.Context, accountID string) (map[string]bool, error) {
	query, args, err := database.QB.
		Select("source_account_id", "target_account_id", "relation_type").
		From("confirmed_relationships").
		Where("source_account_id = ? OR target_account_id = ?", accountID, accountID).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build confirmed relationships query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query confirmed relationships: %w", err)
	}
	defer rows.Close()

	// Key format: "sourceID:targetID:type"
	confirmed := make(map[string]bool)
	for rows.Next() {
		var sourceID, targetID, relType string
		if err := rows.Scan(&sourceID, &targetID, &relType); err != nil {
			return nil, fmt.Errorf("scan confirmed relationship: %w", err)
		}
		key := fmt.Sprintf("%s:%s:%s", sourceID, targetID, relType)
		confirmed[key] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate confirmed relationship rows: %w", err)
	}
	return confirmed, nil
}

// AccountInfo contains account data from the database.
type AccountInfo struct {
	TotalXLMValue float64
	MTLACBalance  float64
}

// LPShareRow represents a liquidity pool share from the database.
type LPShareRow struct {
	PoolID         string
	ShareBalance   float64
	TotalShares    float64
	ReserveACode   string
	ReserveAIssuer string
	ReserveAAmount float64
	ReserveBCode   string
	ReserveBIssuer string
	ReserveBAmount float64
	XLMValue       float64
}

// GetLPShares returns liquidity pool shares for an account.
func (r *AccountRepository) GetLPShares(ctx context.Context, accountID string) ([]LPShareRow, error) {
	query := `
		SELECT
			als.pool_id,
			als.share_balance,
			lp.total_shares,
			lp.reserve_a_code,
			lp.reserve_a_issuer,
			lp.reserve_a_amount,
			lp.reserve_b_code,
			lp.reserve_b_issuer,
			lp.reserve_b_amount,
			COALESCE(als.xlm_value, 0) AS xlm_value
		FROM account_lp_shares als
		JOIN liquidity_pools lp ON als.pool_id = lp.pool_id
		WHERE als.account_id = $1
		ORDER BY COALESCE(als.xlm_value, 0) DESC
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query LP shares: %w", err)
	}
	defer rows.Close()

	var shares []LPShareRow
	for rows.Next() {
		var s LPShareRow
		if err := rows.Scan(
			&s.PoolID,
			&s.ShareBalance,
			&s.TotalShares,
			&s.ReserveACode,
			&s.ReserveAIssuer,
			&s.ReserveAAmount,
			&s.ReserveBCode,
			&s.ReserveBIssuer,
			&s.ReserveBAmount,
			&s.XLMValue,
		); err != nil {
			return nil, fmt.Errorf("scan LP share: %w", err)
		}
		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate LP shares: %w", err)
	}

	return shares, nil
}

// GetAccountInfo returns account information from the database.
func (r *AccountRepository) GetAccountInfo(ctx context.Context, accountID string) (*AccountInfo, error) {
	query, args, err := database.QB.
		Select("COALESCE(total_xlm_value, 0)", "COALESCE(mtlac_balance, 0)").
		From("accounts").
		Where("account_id = ?", accountID).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build account info query: %w", err)
	}

	var info AccountInfo
	err = r.pool.QueryRow(ctx, query, args...).Scan(&info.TotalXLMValue, &info.MTLACBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &AccountInfo{}, nil
		}
		return nil, fmt.Errorf("query account info: %w", err)
	}

	return &info, nil
}

// GetAccountNames returns a map of account IDs to names for the given IDs.
// Accounts not found in the database will not be included in the result.
func (r *AccountRepository) GetAccountNames(ctx context.Context, accountIDs []string) (map[string]string, error) {
	if len(accountIDs) == 0 {
		return make(map[string]string), nil
	}

	query, args, err := database.QB.
		Select("account_id", "name").
		From("accounts").
		Where(sq.Eq{"account_id": accountIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build account names query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query account names: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan account name: %w", err)
		}
		if name != "" {
			result[id] = name
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account names: %w", err)
	}

	return result, nil
}

// TagRow represents a tag with its usage count.
type TagRow struct {
	TagName string
	Count   int
}

// GetAllTags returns all unique tags with their account counts.
func (r *AccountRepository) GetAllTags(ctx context.Context) ([]TagRow, error) {
	query := `
		SELECT SUBSTRING(data_key FROM 4) AS tag_name, COUNT(DISTINCT account_id) AS account_count
		FROM account_metadata
		WHERE data_key LIKE 'Tag%' AND LENGTH(data_key) > 3
		GROUP BY data_key
		ORDER BY account_count DESC, tag_name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all tags: %w", err)
	}
	defer rows.Close()

	var tags []TagRow
	for rows.Next() {
		var tag TagRow
		if err := rows.Scan(&tag.TagName, &tag.Count); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

// SearchAccountRow represents an account from a search query.
type SearchAccountRow struct {
	AccountID        string
	Name             string
	MTLAPBalance     float64
	MTLACBalance     float64
	MTLAXBalance     float64
	TotalXLMValue    float64
	ReputationScore  float64 // Weighted reputation score (0 if no ratings)
	ReputationWeight float64 // Total weight of raters
}

// SearchSortOrder defines sorting options for search results.
type SearchSortOrder string

const (
	SearchSortByBalance    SearchSortOrder = "balance"
	SearchSortByReputation SearchSortOrder = "reputation"
)

// escapeLikePattern escapes special LIKE pattern characters (%, _, \) to prevent
// users from injecting wildcards into search queries.
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// SearchAccounts searches accounts by name or account ID with case-insensitive substring matching.
// If tags are provided, accounts must have ALL specified tags (AND logic).
// Tags should be provided without the "Tag" prefix (e.g., "Belgrade", not "TagBelgrade").
// sortBy specifies the sorting order: "balance" (default) or "reputation".
func (r *AccountRepository) SearchAccounts(ctx context.Context, query string, tags []string, limit int, offset int, sortBy SearchSortOrder) ([]SearchAccountRow, error) {
	// If both query and tags are empty, return nothing
	if query == "" && len(tags) == 0 {
		return nil, nil
	}

	// Base query builder
	qb := database.QB.
		Select(
			"a.account_id",
			"COALESCE(m.data_value, CONCAT(LEFT(a.account_id, 6), '...', RIGHT(a.account_id, 6))) AS name",
			"a.mtlap_balance",
			"a.mtlac_balance",
			"COALESCE(a.mtlax_balance, 0) AS mtlax_balance",
			"a.total_xlm_value",
			"COALESCE(rs.weighted_score, 0) AS reputation_score",
			"COALESCE(rs.total_weight, 0) AS reputation_weight",
		).
		From("accounts a").
		LeftJoin("account_metadata m ON a.account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''").
		LeftJoin("reputation_scores rs ON a.account_id = rs.account_id")

	// Add text search condition if query provided
	if query != "" {
		searchPattern := "%" + escapeLikePattern(query) + "%"
		qb = qb.Where(sq.Or{
			sq.ILike{"a.account_id": searchPattern},
			sq.ILike{"m.data_value": searchPattern},
			sq.ILike{"a.name": searchPattern},
		})
	}

	// Add tag filter condition if tags provided
	if len(tags) > 0 {
		tagKeys := lo.Map(tags, func(t string, _ int) string {
			return "Tag" + t
		})

		// Use JOIN with aggregation instead of subquery for proper placeholder handling
		qb = qb.
			Join("account_metadata tags ON a.account_id = tags.account_id").
			Where(sq.Eq{"tags.data_key": tagKeys}).
			GroupBy("a.account_id", "m.data_value", "rs.weighted_score", "rs.total_weight").
			Having(fmt.Sprintf("COUNT(DISTINCT tags.data_key) = %d", len(tagKeys)))
	}

	// Apply sorting
	switch sortBy {
	case SearchSortByReputation:
		// Sort by membership level first (MTLAP/MTLAC balance), then by grade bucket, then by weight
		qb = qb.OrderBy(
			"GREATEST(a.mtlap_balance, a.mtlac_balance, COALESCE(a.mtlax_balance, 0)) DESC",
			`CASE
				WHEN COALESCE(rs.weighted_score, 0) >= 3.5 THEN 1
				WHEN COALESCE(rs.weighted_score, 0) >= 3.0 THEN 2
				WHEN COALESCE(rs.weighted_score, 0) >= 2.5 THEN 3
				WHEN COALESCE(rs.weighted_score, 0) >= 2.0 THEN 4
				WHEN COALESCE(rs.weighted_score, 0) >= 1.5 THEN 5
				WHEN COALESCE(rs.weighted_score, 0) >= 1.0 THEN 6
				WHEN COALESCE(rs.weighted_score, 0) > 0 THEN 7
				ELSE 8
			END ASC`,
			"COALESCE(rs.total_weight, 0) DESC",
		)
	default: // SearchSortByBalance
		qb = qb.OrderBy("GREATEST(a.mtlap_balance, a.mtlac_balance, COALESCE(a.mtlax_balance, 0)) DESC", "a.total_xlm_value DESC")
	}

	qb = qb.Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query search accounts: %w", err)
	}
	defer rows.Close()

	var accounts []SearchAccountRow
	for rows.Next() {
		var acc SearchAccountRow
		if err := rows.Scan(&acc.AccountID, &acc.Name, &acc.MTLAPBalance, &acc.MTLACBalance, &acc.MTLAXBalance, &acc.TotalXLMValue, &acc.ReputationScore, &acc.ReputationWeight); err != nil {
			return nil, fmt.Errorf("scan search account: %w", err)
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search accounts: %w", err)
	}

	return accounts, nil
}

// CountSearchAccounts returns the total count of accounts matching the search query and/or tags.
// If tags are provided, accounts must have ALL specified tags (AND logic).
// Tags should be provided without the "Tag" prefix (e.g., "Belgrade", not "TagBelgrade").
func (r *AccountRepository) CountSearchAccounts(ctx context.Context, query string, tags []string) (int, error) {
	// If both query and tags are empty, return 0
	if query == "" && len(tags) == 0 {
		return 0, nil
	}

	// Base query builder - use COUNT(DISTINCT) to handle potential JOINs
	qb := database.QB.
		Select("COUNT(DISTINCT a.account_id)").
		From("accounts a").
		LeftJoin("account_metadata m ON a.account_id = m.account_id AND m.data_key = 'Name' AND m.data_index = ''")

	// Add text search condition if query provided
	if query != "" {
		searchPattern := "%" + escapeLikePattern(query) + "%"
		qb = qb.Where(sq.Or{
			sq.ILike{"a.account_id": searchPattern},
			sq.ILike{"m.data_value": searchPattern},
			sq.ILike{"a.name": searchPattern},
		})
	}

	// Add tag filter condition if tags provided
	if len(tags) > 0 {
		tagKeys := lo.Map(tags, func(t string, _ int) string {
			return "Tag" + t
		})

		// Use JOIN with aggregation instead of subquery for proper placeholder handling
		qb = qb.
			Join("account_metadata tags ON a.account_id = tags.account_id").
			Where(sq.Eq{"tags.data_key": tagKeys}).
			GroupBy("a.account_id").
			Having(fmt.Sprintf("COUNT(DISTINCT tags.data_key) = %d", len(tagKeys)))
	}

	// When using GROUP BY with HAVING, wrap in subquery to get total count
	if len(tags) > 0 {
		innerSQL, innerArgs, err := qb.ToSql()
		if err != nil {
			return 0, fmt.Errorf("build inner count search query: %w", err)
		}

		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS filtered", innerSQL)
		var count int
		err = r.pool.QueryRow(ctx, countSQL, innerArgs...).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("query count search accounts: %w", err)
		}
		return count, nil
	}

	sql, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count search query: %w", err)
	}

	var count int
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query count search accounts: %w", err)
	}

	return count, nil
}
