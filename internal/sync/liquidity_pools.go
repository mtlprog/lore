package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
)

// syncLPShares syncs liquidity pool shares for an account.
// It parses LP shares from the account balances, fetches pool details,
// and stores them in the database.
func (s *Syncer) syncLPShares(ctx context.Context, accountID string, acc *horizon.Account) error {
	// Find LP share balances
	var lpShares []LPShare
	for _, bal := range acc.Balances {
		if bal.Type != "liquidity_pool_shares" {
			continue
		}

		balance, err := decimal.NewFromString(bal.Balance)
		if err != nil {
			s.logger.Warn("failed to parse LP share balance", "account", accountID, "error", err)
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

	// Delete existing LP shares for this account (for re-sync)
	if err := s.repo.DeleteAccountLPShares(ctx, accountID); err != nil {
		return fmt.Errorf("delete existing LP shares: %w", err)
	}

	if len(lpShares) == 0 {
		return nil
	}

	// Fetch pool details and store shares
	for _, share := range lpShares {
		pool, err := s.fetchLPPoolDetail(ctx, share.PoolID)
		if err != nil {
			s.logger.Warn("failed to fetch LP pool detail", "pool_id", share.PoolID, "error", err)
			continue
		}

		// Upsert pool data
		if err := s.repo.UpsertLPPool(ctx, pool); err != nil {
			s.logger.Warn("failed to upsert LP pool", "pool_id", pool.PoolID, "error", err)
			continue
		}

		// Upsert share data
		if err := s.repo.UpsertLPShare(ctx, accountID, share.PoolID, share.ShareBalance); err != nil {
			s.logger.Warn("failed to upsert LP share", "account", accountID, "pool_id", share.PoolID, "error", err)
			continue
		}
	}

	return nil
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
