package sync

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/config"
	"github.com/stellar/go/clients/horizonclient"
)

// Syncer orchestrates the synchronization of Stellar data to PostgreSQL.
type Syncer struct {
	horizon *horizonclient.Client
	repo    *Repository
	logger  *slog.Logger
}

// New creates a new Syncer instance.
func New(pool *pgxpool.Pool, horizonURL string) *Syncer {
	return &Syncer{
		horizon: &horizonclient.Client{HorizonURL: horizonURL},
		repo:    NewRepository(pool),
		logger:  slog.Default(),
	}
}

// Run executes the full synchronization process.
func (s *Syncer) Run(ctx context.Context, full bool) error {
	s.logger.Info("starting sync", "full", full)

	if full {
		s.logger.Info("truncating tables for full sync")
		if err := s.repo.Truncate(ctx); err != nil {
			return fmt.Errorf("truncate tables: %w", err)
		}
	}

	// Step 1: Collect all unique account IDs from MTLAP and MTLAC holders
	s.logger.Info("fetching MTLAP holders")
	mtlapHolders, err := s.fetchAllAssetHolders(ctx, config.TokenMTLAP, config.TokenIssuer)
	if err != nil {
		return fmt.Errorf("fetch MTLAP holders: %w", err)
	}
	s.logger.Info("fetched MTLAP holders", "count", len(mtlapHolders))

	s.logger.Info("fetching MTLAC holders")
	mtlacHolders, err := s.fetchAllAssetHolders(ctx, config.TokenMTLAC, config.TokenIssuer)
	if err != nil {
		return fmt.Errorf("fetch MTLAC holders: %w", err)
	}
	s.logger.Info("fetched MTLAC holders", "count", len(mtlacHolders))

	// Merge into unique set
	accountIDs := make(map[string]struct{})
	for _, id := range mtlapHolders {
		accountIDs[id] = struct{}{}
	}
	for _, id := range mtlacHolders {
		accountIDs[id] = struct{}{}
	}
	s.logger.Info("unique accounts to sync", "count", len(accountIDs))

	// Step 2: Fetch and store details for each account
	s.logger.Info("fetching account details")
	if err := s.syncAccounts(ctx, accountIDs); err != nil {
		return fmt.Errorf("sync accounts: %w", err)
	}

	// Step 3: Fetch token prices from SDEX
	s.logger.Info("fetching token prices")
	if err := s.syncTokenPrices(ctx); err != nil {
		return fmt.Errorf("sync token prices: %w", err)
	}

	// Step 4: Update XLM values based on prices
	s.logger.Info("updating XLM values")
	if err := s.repo.UpdateXLMValues(ctx); err != nil {
		return fmt.Errorf("update XLM values: %w", err)
	}

	// Step 5: Calculate delegations
	s.logger.Info("calculating delegations")
	if err := s.calculateDelegations(ctx); err != nil {
		return fmt.Errorf("calculate delegations: %w", err)
	}

	// Step 6: Fetch association tags
	s.logger.Info("fetching association tags")
	if err := s.syncAssociationTags(ctx); err != nil {
		return fmt.Errorf("sync association tags: %w", err)
	}

	// Log summary
	stats, err := s.repo.GetSyncStats(ctx)
	if err != nil {
		s.logger.Warn("failed to get sync stats", "error", err)
	} else {
		s.logger.Info("sync completed",
			"accounts", stats.TotalAccounts,
			"persons", stats.TotalPersons,
			"companies", stats.TotalCompanies,
		)
	}

	return nil
}
