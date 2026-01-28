package sync

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/database"
	"github.com/shopspring/decimal"
)

// Repository handles database operations for syncing.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new sync repository.
// Returns error if pool is nil.
func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("database pool is required")
	}
	return &Repository{pool: pool}, nil
}

// allowedTruncateTables is the whitelist of tables that can be truncated.
var allowedTruncateTables = map[string]bool{
	"association_tags": true,
	"relationships":    true,
	"account_metadata": true,
	"account_balances": true,
	"token_prices":     true,
	"accounts":         true,
}

// Truncate clears all syncable tables (preserves settings tables).
func (r *Repository) Truncate(ctx context.Context) error {
	tables := []string{
		"association_tags",
		"relationships",
		"account_metadata",
		"account_balances",
		"token_prices",
		"accounts",
	}

	for _, table := range tables {
		if !allowedTruncateTables[table] {
			return fmt.Errorf("table %q is not allowed to be truncated", table)
		}
		_, err := r.pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %q CASCADE", table))
		if err != nil {
			return fmt.Errorf("truncate %q: %w", table, err)
		}
	}

	return nil
}

// UpsertAccount inserts or updates an account.
func (r *Repository) UpsertAccount(ctx context.Context, data *AccountData) error {
	mtlapBalance := getMTLAPBalance(data)
	mtlacBalance := getMTLACBalance(data)
	nativeBalance := getNativeBalance(data)

	query, args, err := database.QB.
		Insert("accounts").
		Columns(
			"account_id",
			"mtlap_balance",
			"mtlac_balance",
			"native_balance",
			"delegate_to",
			"is_council_ready",
			"updated_at",
		).
		Values(
			data.ID,
			mtlapBalance,
			mtlacBalance,
			nativeBalance,
			data.DelegateTo,
			data.CouncilReady,
			"NOW()",
		).
		Suffix(`ON CONFLICT (account_id) DO UPDATE SET
			mtlap_balance = EXCLUDED.mtlap_balance,
			mtlac_balance = EXCLUDED.mtlac_balance,
			native_balance = EXCLUDED.native_balance,
			delegate_to = EXCLUDED.delegate_to,
			is_council_ready = EXCLUDED.is_council_ready,
			updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}

	return nil
}

// UpsertBalances inserts or updates account balances within a transaction.
func (r *Repository) UpsertBalances(ctx context.Context, accountID string, balances []Balance) error {
	if len(balances) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM account_balances WHERE account_id = $1", accountID)
	if err != nil {
		return fmt.Errorf("delete existing balances: %w", err)
	}

	query := database.QB.Insert("account_balances").
		Columns("account_id", "asset_code", "asset_issuer", "balance")

	for _, bal := range balances {
		query = query.Values(accountID, bal.AssetCode, bal.AssetIssuer, bal.Balance)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// UpsertMetadata inserts or updates account metadata within a transaction.
func (r *Repository) UpsertMetadata(ctx context.Context, accountID string, metadata []Metadata) error {
	if len(metadata) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM account_metadata WHERE account_id = $1", accountID)
	if err != nil {
		return fmt.Errorf("delete existing metadata: %w", err)
	}

	query := database.QB.Insert("account_metadata").
		Columns("account_id", "data_key", "data_index", "data_value")

	for _, m := range metadata {
		query = query.Values(accountID, m.Key, strconv.Itoa(m.Index), m.Value)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// UpsertRelationships inserts or updates account relationships within a transaction.
func (r *Repository) UpsertRelationships(ctx context.Context, accountID string, relationships []Relationship) error {
	if len(relationships) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM relationships WHERE source_account_id = $1", accountID)
	if err != nil {
		return fmt.Errorf("delete existing relationships: %w", err)
	}

	query := database.QB.Insert("relationships").
		Columns("source_account_id", "target_account_id", "relation_type", "relation_index")

	for _, rel := range relationships {
		query = query.Values(accountID, rel.TargetAccountID, rel.RelationType, strconv.Itoa(rel.RelationIndex))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetUniqueAssets returns all unique assets from account_balances.
func (r *Repository) GetUniqueAssets(ctx context.Context) ([]Asset, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT asset_code, asset_issuer
		FROM account_balances
		ORDER BY asset_code
	`)
	if err != nil {
		return nil, fmt.Errorf("query unique assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.Code, &asset.Issuer); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}

	return assets, nil
}

// UpsertTokenPrice inserts or updates a token price.
func (r *Repository) UpsertTokenPrice(ctx context.Context, code, issuer string, price decimal.Decimal) error {
	query, args, err := database.QB.
		Insert("token_prices").
		Columns("asset_code", "asset_issuer", "xlm_price", "updated_at").
		Values(code, issuer, price, "NOW()").
		Suffix(`ON CONFLICT (asset_code, asset_issuer) DO UPDATE SET
			xlm_price = EXCLUDED.xlm_price,
			updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}

	return nil
}

// UpdateXLMValues calculates and updates total_xlm_value for all accounts.
func (r *Repository) UpdateXLMValues(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE account_balances ab
		SET xlm_value = ab.balance * COALESCE(tp.xlm_price, 0)
		FROM token_prices tp
		WHERE ab.asset_code = tp.asset_code
		  AND ab.asset_issuer = tp.asset_issuer
	`)
	if err != nil {
		return fmt.Errorf("update balance xlm_values: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE accounts a
		SET total_xlm_value = COALESCE((
			SELECT SUM(xlm_value)
			FROM account_balances ab
			WHERE ab.account_id = a.account_id
		), 0)
	`)
	if err != nil {
		return fmt.Errorf("update account total_xlm_values: %w", err)
	}

	return nil
}

// ResetDelegations resets all delegation-related fields.
func (r *Repository) ResetDelegations(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE accounts SET
			received_votes = 0,
			has_delegation_error = FALSE,
			has_cycle_error = FALSE,
			cycle_path = NULL
	`)
	if err != nil {
		return fmt.Errorf("reset delegations: %w", err)
	}
	return nil
}

// GetAllDelegationInfo returns delegation info for all accounts.
func (r *Repository) GetAllDelegationInfo(ctx context.Context) ([]DelegationInfo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT account_id, delegate_to, mtlap_balance, is_council_ready
		FROM accounts
	`)
	if err != nil {
		return nil, fmt.Errorf("query delegation info: %w", err)
	}
	defer rows.Close()

	var infos []DelegationInfo
	for rows.Next() {
		var info DelegationInfo
		if err := rows.Scan(&info.AccountID, &info.DelegateTo, &info.MTLAPBalance, &info.CouncilReady); err != nil {
			return nil, fmt.Errorf("scan delegation info: %w", err)
		}
		infos = append(infos, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delegation info: %w", err)
	}

	return infos, nil
}

// SetDelegationError marks an account as having a delegation error.
func (r *Repository) SetDelegationError(ctx context.Context, accountID string, hasError bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE accounts SET has_delegation_error = $1 WHERE account_id = $2
	`, hasError, accountID)
	if err != nil {
		return fmt.Errorf("set delegation error: %w", err)
	}
	return nil
}

// SetCycleError marks an account as having a cycle error.
func (r *Repository) SetCycleError(ctx context.Context, accountID string, path []string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE accounts SET has_cycle_error = TRUE, cycle_path = $1 WHERE account_id = $2
	`, path, accountID)
	if err != nil {
		return fmt.Errorf("set cycle error: %w", err)
	}
	return nil
}

// SetReceivedVotes updates the received_votes for an account.
func (r *Repository) SetReceivedVotes(ctx context.Context, accountID string, votes int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE accounts SET received_votes = $1 WHERE account_id = $2
	`, votes, accountID)
	if err != nil {
		return fmt.Errorf("set received votes: %w", err)
	}
	return nil
}

// GetSyncStats returns aggregate statistics.
func (r *Repository) GetSyncStats(ctx context.Context) (*SyncStats, error) {
	var stats SyncStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_accounts,
			COUNT(*) FILTER (WHERE mtlap_balance > 0) AS total_persons,
			COUNT(*) FILTER (WHERE mtlac_balance > 0) AS total_companies,
			COALESCE(SUM(total_xlm_value), 0) AS total_xlm_value
		FROM accounts
	`).Scan(&stats.TotalAccounts, &stats.TotalPersons, &stats.TotalCompanies, &stats.TotalXLMValue)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}
	return &stats, nil
}

// UpsertAssociationTags inserts or updates association tags within a transaction.
func (r *Repository) UpsertAssociationTags(ctx context.Context, tagName TagName, tags []AssociationTag) error {
	if len(tags) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM association_tags WHERE tag_name = $1", tagName)
	if err != nil {
		return fmt.Errorf("delete existing tags: %w", err)
	}

	query := database.QB.Insert("association_tags").
		Columns("tag_name", "tag_index", "target_account_id")

	for _, tag := range tags {
		query = query.Values(tag.TagName, strconv.Itoa(tag.TagIndex), tag.TargetAccountID)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
