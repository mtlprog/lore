package reputation

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/database"
	"github.com/shopspring/decimal"
)

// Repository handles reputation data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new reputation repository.
func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("database pool is required")
	}
	return &Repository{pool: pool}, nil
}

// GetRatingEdges returns all A/B/C/D relationships.
func (r *Repository) GetRatingEdges(ctx context.Context) ([]RatingEdge, error) {
	query, args, err := database.QB.
		Select("source_account_id", "target_account_id", "relation_type").
		From("relationships").
		Where(sq.Eq{"relation_type": []string{"A", "B", "C", "D"}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build rating edges query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rating edges: %w", err)
	}
	defer rows.Close()

	var edges []RatingEdge
	for rows.Next() {
		var e RatingEdge
		if err := rows.Scan(&e.RaterAccountID, &e.RateeAccountID, &e.Rating); err != nil {
			return nil, fmt.Errorf("scan rating edge: %w", err)
		}
		edges = append(edges, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rating edges: %w", err)
	}

	return edges, nil
}

// GetPortfolios returns total_xlm_value for all accounts.
func (r *Repository) GetPortfolios(ctx context.Context) (map[string]decimal.Decimal, error) {
	query, args, err := database.QB.
		Select("account_id", "COALESCE(total_xlm_value, 0)").
		From("accounts").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build portfolios query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query portfolios: %w", err)
	}
	defer rows.Close()

	result := make(map[string]decimal.Decimal)
	for rows.Next() {
		var accountID string
		var value decimal.Decimal
		if err := rows.Scan(&accountID, &value); err != nil {
			return nil, fmt.Errorf("scan portfolio: %w", err)
		}
		result[accountID] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portfolios: %w", err)
	}

	return result, nil
}

// GetConnectionCounts returns count of confirmed relationships per account.
func (r *Repository) GetConnectionCounts(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT account_id, COUNT(*) AS connection_count
		FROM (
			SELECT source_account_id AS account_id FROM confirmed_relationships
			UNION ALL
			SELECT target_account_id AS account_id FROM confirmed_relationships
		) connections
		GROUP BY account_id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query connection counts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var accountID string
		var count int
		if err := rows.Scan(&accountID, &count); err != nil {
			return nil, fmt.Errorf("scan connection count: %w", err)
		}
		result[accountID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connection counts: %w", err)
	}

	return result, nil
}

// UpsertScores bulk upserts reputation scores.
// Only scores for accounts that exist in the accounts table will be saved
// (foreign key constraint requires this).
func (r *Repository) UpsertScores(ctx context.Context, scores map[string]*Score) error {
	if len(scores) == 0 {
		return nil
	}

	// Get all account IDs we want to insert
	accountIDs := make([]string, 0, len(scores))
	for id := range scores {
		accountIDs = append(accountIDs, id)
	}

	// Query which accounts actually exist in the database
	query, args, err := database.QB.
		Select("account_id").
		From("accounts").
		Where(sq.Eq{"account_id": accountIDs}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build existing accounts query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query existing accounts: %w", err)
	}

	existingAccounts := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing account: %w", err)
		}
		existingAccounts[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate existing accounts: %w", err)
	}

	// Use batch for performance
	batch := &pgx.Batch{}
	now := time.Now()
	insertCount := 0

	for _, score := range scores {
		// Skip accounts that don't exist in the accounts table
		if !existingAccounts[score.AccountID] {
			continue
		}
		insertCount++
		query := `
			INSERT INTO reputation_scores (
				account_id, weighted_score, base_score,
				rating_count_a, rating_count_b, rating_count_c, rating_count_d,
				total_ratings, total_weight, calculated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (account_id) DO UPDATE SET
				weighted_score = EXCLUDED.weighted_score,
				base_score = EXCLUDED.base_score,
				rating_count_a = EXCLUDED.rating_count_a,
				rating_count_b = EXCLUDED.rating_count_b,
				rating_count_c = EXCLUDED.rating_count_c,
				rating_count_d = EXCLUDED.rating_count_d,
				total_ratings = EXCLUDED.total_ratings,
				total_weight = EXCLUDED.total_weight,
				calculated_at = EXCLUDED.calculated_at
		`
		batch.Queue(query,
			score.AccountID,
			score.WeightedScore,
			score.BaseScore,
			score.RatingCountA,
			score.RatingCountB,
			score.RatingCountC,
			score.RatingCountD,
			score.TotalRatings,
			score.TotalWeight,
			now,
		)
	}

	if insertCount == 0 {
		return nil
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range insertCount {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("execute batch: %w", err)
		}
	}

	return nil
}

// GetScore returns the reputation score for an account.
func (r *Repository) GetScore(ctx context.Context, accountID string) (*Score, error) {
	query, args, err := database.QB.
		Select(
			"account_id", "weighted_score", "base_score",
			"rating_count_a", "rating_count_b", "rating_count_c", "rating_count_d",
			"total_ratings", "total_weight", "calculated_at",
		).
		From("reputation_scores").
		Where(sq.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build score query: %w", err)
	}

	var score Score
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&score.AccountID,
		&score.WeightedScore,
		&score.BaseScore,
		&score.RatingCountA,
		&score.RatingCountB,
		&score.RatingCountC,
		&score.RatingCountD,
		&score.TotalRatings,
		&score.TotalWeight,
		&score.CalculatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query score: %w", err)
	}

	return &score, nil
}

// GetDirectRaters returns accounts that gave A/B/C/D ratings to the target account.
func (r *Repository) GetDirectRaters(ctx context.Context, targetAccountID string) ([]RaterInfo, error) {
	query := `
		SELECT
			r.source_account_id,
			COALESCE(NULLIF(a.name, ''), CONCAT(LEFT(r.source_account_id, 6), '...', RIGHT(r.source_account_id, 6))),
			r.relation_type,
			COALESCE(a.total_xlm_value, 0),
			COALESCE(rs.weighted_score, 0)
		FROM relationships r
		LEFT JOIN accounts a ON r.source_account_id = a.account_id
		LEFT JOIN reputation_scores rs ON r.source_account_id = rs.account_id
		WHERE r.target_account_id = $1
		  AND r.relation_type IN ('A', 'B', 'C', 'D')
		ORDER BY r.relation_type, a.name
	`

	rows, err := r.pool.Query(ctx, query, targetAccountID)
	if err != nil {
		return nil, fmt.Errorf("query direct raters: %w", err)
	}
	defer rows.Close()

	var raters []RaterInfo
	for rows.Next() {
		var ri RaterInfo
		if err := rows.Scan(
			&ri.AccountID,
			&ri.Name,
			&ri.Rating,
			&ri.PortfolioXLM,
			&ri.OwnScore,
		); err != nil {
			return nil, fmt.Errorf("scan direct rater: %w", err)
		}
		raters = append(raters, ri)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate direct raters: %w", err)
	}

	return raters, nil
}

// GetRatersOfRaters returns accounts that gave A/B/C/D ratings to the level 1 raters.
func (r *Repository) GetRatersOfRaters(ctx context.Context, level1AccountIDs []string, excludeAccountIDs []string) ([]RaterInfo, error) {
	if len(level1AccountIDs) == 0 {
		return nil, nil
	}

	// Build exclusion set (target account + level 1 accounts to avoid duplicates)
	excludeSet := make(map[string]bool)
	for _, id := range excludeAccountIDs {
		excludeSet[id] = true
	}
	for _, id := range level1AccountIDs {
		excludeSet[id] = true
	}

	query := `
		SELECT DISTINCT ON (r.source_account_id, r.relation_type)
			r.source_account_id,
			COALESCE(NULLIF(a.name, ''), CONCAT(LEFT(r.source_account_id, 6), '...', RIGHT(r.source_account_id, 6))) AS display_name,
			r.relation_type,
			COALESCE(a.total_xlm_value, 0),
			COALESCE(rs.weighted_score, 0),
			r.target_account_id
		FROM relationships r
		LEFT JOIN accounts a ON r.source_account_id = a.account_id
		LEFT JOIN reputation_scores rs ON r.source_account_id = rs.account_id
		WHERE r.target_account_id = ANY($1::text[])
		  AND r.relation_type IN ('A', 'B', 'C', 'D')
		ORDER BY r.source_account_id, r.relation_type
	`

	rows, err := r.pool.Query(ctx, query, level1AccountIDs)
	if err != nil {
		return nil, fmt.Errorf("query raters of raters: %w", err)
	}
	defer rows.Close()

	var raters []RaterInfo
	for rows.Next() {
		var ri RaterInfo
		var targetID string
		if err := rows.Scan(
			&ri.AccountID,
			&ri.Name,
			&ri.Rating,
			&ri.PortfolioXLM,
			&ri.OwnScore,
			&targetID,
		); err != nil {
			return nil, fmt.Errorf("scan rater of rater: %w", err)
		}
		// Skip if in exclusion set
		if excludeSet[ri.AccountID] {
			continue
		}
		raters = append(raters, ri)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate raters of raters: %w", err)
	}

	return raters, nil
}

// GetAccountName returns the name for an account.
func (r *Repository) GetAccountName(ctx context.Context, accountID string) (string, error) {
	query, args, err := database.QB.
		Select("COALESCE(name, CONCAT(LEFT(account_id, 6), '...', RIGHT(account_id, 6)))").
		From("accounts").
		Where(sq.Eq{"account_id": accountID}).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build account name query: %w", err)
	}

	var name string
	err = r.pool.QueryRow(ctx, query, args...).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Truncate account ID if not found
			if len(accountID) > 12 {
				return accountID[:6] + "..." + accountID[len(accountID)-6:], nil
			}
			return accountID, nil
		}
		return "", fmt.Errorf("query account name: %w", err)
	}

	return name, nil
}
