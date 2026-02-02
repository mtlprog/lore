package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
)

// syncLPShares syncs liquidity pool shares for an account.
// It parses LP shares from the account balances, fetches pool details,
// and stores them in the database within a transaction for atomicity.
func (s *Syncer) syncLPShares(ctx context.Context, accountID string, acc *horizon.Account) error {
	// Find LP share balances
	var lpShares []LPShare
	var parseFailures int
	for _, bal := range acc.Balances {
		if bal.Type != "liquidity_pool_shares" {
			continue
		}

		balance, err := decimal.NewFromString(bal.Balance)
		if err != nil {
			s.logger.Error("failed to parse LP share balance", "account", accountID, "pool_id", bal.LiquidityPoolId, "error", err)
			parseFailures++
			continue
		}

		if balance.IsZero() {
			continue
		}

		lpShares = append(lpShares, LPShare{
			PoolID:       bal.LiquidityPoolId,
			ShareBalance: balance,
		})
	}

	// Use transaction for atomicity: delete + inserts should all succeed or all fail
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin LP shares transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Delete existing LP shares for this account (for re-sync)
	if _, err := tx.Exec(ctx, "DELETE FROM account_lp_shares WHERE account_id = $1", accountID); err != nil {
		return fmt.Errorf("delete existing LP shares: %w", err)
	}

	if len(lpShares) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit LP shares transaction: %w", err)
		}
		return nil
	}

	// Fetch pool details and store shares, tracking failures
	var failedPools []string
	for _, share := range lpShares {
		pool, err := s.fetchLPPoolDetail(ctx, share.PoolID)
		if err != nil {
			s.logger.Error("failed to fetch LP pool detail", "account", accountID, "pool_id", share.PoolID, "error", err)
			failedPools = append(failedPools, share.PoolID)
			continue
		}

		// Upsert pool data (outside transaction - pool data is shared across accounts)
		if err := s.repo.UpsertLPPool(ctx, pool); err != nil {
			s.logger.Error("failed to upsert LP pool", "account", accountID, "pool_id", pool.PoolID, "error", err)
			failedPools = append(failedPools, share.PoolID)
			continue
		}

		// Upsert share data within transaction
		if err := upsertLPShareTx(ctx, tx, accountID, share.PoolID, share.ShareBalance); err != nil {
			s.logger.Error("failed to upsert LP share", "account", accountID, "pool_id", share.PoolID, "error", err)
			failedPools = append(failedPools, share.PoolID)
			continue
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit LP shares transaction: %w", err)
	}

	// Log summary if there were failures (but don't fail the whole sync)
	if len(failedPools) > 0 || parseFailures > 0 {
		s.logger.Error("LP share sync completed with failures",
			"account", accountID,
			"failed_pools", failedPools,
			"parse_failures", parseFailures,
			"successful", len(lpShares)-len(failedPools),
		)
	}

	return nil
}

// upsertLPShareTx inserts or updates an LP share within a transaction.
func upsertLPShareTx(ctx context.Context, tx pgx.Tx, accountID, poolID string, balance decimal.Decimal) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO account_lp_shares (account_id, pool_id, share_balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id, pool_id) DO UPDATE SET share_balance = EXCLUDED.share_balance
	`, accountID, poolID, balance)
	return err
}

// fetchLPPoolDetail fetches liquidity pool details from Horizon.
func (s *Syncer) fetchLPPoolDetail(ctx context.Context, poolID string) (*LPPoolData, error) {
	pool, err := s.horizon.LiquidityPoolDetail(horizonclient.LiquidityPoolRequest{
		LiquidityPoolID: poolID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch liquidity pool: %w", err)
	}

	if len(pool.Reserves) < 2 {
		return nil, fmt.Errorf("unexpected reserve count: %d", len(pool.Reserves))
	}

	totalShares, err := decimal.NewFromString(pool.TotalShares)
	if err != nil {
		return nil, fmt.Errorf("parse total shares: %w", err)
	}

	reserveACode, reserveAIssuer := parseReserveAsset(pool.Reserves[0].Asset)
	reserveAAmount, err := decimal.NewFromString(pool.Reserves[0].Amount)
	if err != nil {
		return nil, fmt.Errorf("parse reserve A amount: %w", err)
	}

	reserveBCode, reserveBIssuer := parseReserveAsset(pool.Reserves[1].Asset)
	reserveBAmount, err := decimal.NewFromString(pool.Reserves[1].Amount)
	if err != nil {
		return nil, fmt.Errorf("parse reserve B amount: %w", err)
	}

	return &LPPoolData{
		PoolID:         pool.ID,
		TotalShares:    totalShares,
		ReserveACode:   reserveACode,
		ReserveAIssuer: reserveAIssuer,
		ReserveAAmount: reserveAAmount,
		ReserveBCode:   reserveBCode,
		ReserveBIssuer: reserveBIssuer,
		ReserveBAmount: reserveBAmount,
	}, nil
}

// parseReserveAsset parses asset from "CODE:ISSUER" format.
// Native XLM is represented as "native".
func parseReserveAsset(asset string) (code, issuer string) {
	if asset == "native" {
		return "XLM", ""
	}

	parts := strings.SplitN(asset, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return asset, ""
}
