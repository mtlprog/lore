package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/database"
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

// GetStats returns aggregate statistics.
func (r *AccountRepository) GetStats(ctx context.Context) (*Stats, error) {
	query, args, err := database.QB.
		Select(
			"COUNT(*) AS total_accounts",
			"COUNT(*) FILTER (WHERE mtlap_balance > 0 AND mtlap_balance <= 5) AS total_persons",
			"COUNT(*) FILTER (WHERE mtlac_balance > 0 AND mtlac_balance <= 4) AS total_companies",
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

// IsConfirmed checks if a relationship has a confirmed pair (e.g., MyPart/PartOf).
func (r *AccountRepository) IsConfirmed(ctx context.Context, sourceID, targetID, relationType string) (bool, error) {
	// Check if the confirmed_relationships view has this pair
	query, args, err := database.QB.
		Select("1").
		From("confirmed_relationships").
		Where("source_account_id = ?", sourceID).
		Where("target_account_id = ?", targetID).
		Where("relation_type = ?", relationType).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build confirmed check query: %w", err)
	}

	var exists int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("query confirmed check: %w", err)
	}
	return true, nil
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
		if err.Error() == "no rows in result set" {
			return &AccountInfo{}, nil
		}
		return nil, fmt.Errorf("query account info: %w", err)
	}

	return &info, nil
}
