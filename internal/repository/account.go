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
