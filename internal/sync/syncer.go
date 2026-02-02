package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/reputation"
	"github.com/samber/lo"
	"github.com/stellar/go/clients/horizonclient"
)

// Syncer orchestrates the synchronization of Stellar data to PostgreSQL.
type Syncer struct {
	horizon          *horizonclient.Client
	repo             *Repository
	logger           *slog.Logger
	failureThreshold float64
}

// SyncerOption is a functional option for configuring a Syncer.
type SyncerOption func(*Syncer)

// WithFailureThreshold sets the maximum failure rate before sync is considered failed.
// Default is 0.1 (10%). The threshold is expressed as a fraction (0.0-1.0).
func WithFailureThreshold(threshold float64) SyncerOption {
	return func(s *Syncer) {
		s.failureThreshold = threshold
	}
}

// WithLogger sets a custom logger for the syncer.
func WithLogger(logger *slog.Logger) SyncerOption {
	return func(s *Syncer) {
		s.logger = logger
	}
}

// New creates a new Syncer instance.
// Returns error if pool is nil or horizonURL is empty.
func New(pool *pgxpool.Pool, horizonURL string, opts ...SyncerOption) (*Syncer, error) {
	if horizonURL == "" {
		return nil, errors.New("horizon URL is required")
	}

	repo, err := NewRepository(pool)
	if err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	s := &Syncer{
		horizon:          &horizonclient.Client{HorizonURL: horizonURL},
		repo:             repo,
		logger:           slog.Default(),
		failureThreshold: DefaultFailureThreshold,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Run executes the full synchronization process.
// Returns SyncResult with statistics and any failures encountered.
func (s *Syncer) Run(ctx context.Context, full bool) (*SyncResult, error) {
	s.logger.Info("starting sync", "full", full)

	if full {
		s.logger.Info("truncating tables for full sync")
		if err := s.repo.Truncate(ctx); err != nil {
			return nil, fmt.Errorf("truncate tables: %w", err)
		}
	}

	// Step 1: Collect all unique account IDs from MTLAP and MTLAC holders
	s.logger.Info("fetching MTLAP holders")
	mtlapHolders, err := s.fetchAllAssetHolders(ctx, config.TokenMTLAP, config.TokenIssuer)
	if err != nil {
		return nil, fmt.Errorf("fetch MTLAP holders: %w", err)
	}
	s.logger.Info("fetched MTLAP holders", "count", len(mtlapHolders))

	s.logger.Info("fetching MTLAC holders")
	mtlacHolders, err := s.fetchAllAssetHolders(ctx, config.TokenMTLAC, config.TokenIssuer)
	if err != nil {
		return nil, fmt.Errorf("fetch MTLAC holders: %w", err)
	}
	s.logger.Info("fetched MTLAC holders", "count", len(mtlacHolders))

	// Merge into unique list using lo.Uniq
	accountIDs := lo.Uniq(append(mtlapHolders, mtlacHolders...))
	s.logger.Info("unique accounts to sync", "count", len(accountIDs))

	// Step 2: Fetch and store details for each account
	s.logger.Info("fetching account details")
	result, err := s.syncAccounts(ctx, accountIDs)
	if err != nil {
		return result, fmt.Errorf("sync accounts: %w", err)
	}

	// Step 3: Fetch token prices from SDEX
	s.logger.Info("fetching token prices")
	failedPrices, err := s.syncTokenPrices(ctx)
	if err != nil {
		result.FailedPrices = failedPrices
		return result, fmt.Errorf("sync token prices: %w", err)
	}
	result.FailedPrices = failedPrices

	// Step 4: Update XLM values based on prices (including LP shares)
	s.logger.Info("updating LP share values")
	if err := s.repo.UpdateLPShareValues(ctx); err != nil {
		return result, fmt.Errorf("update LP share values: %w", err)
	}

	s.logger.Info("updating XLM values")
	if err := s.repo.UpdateXLMValues(ctx); err != nil {
		return result, fmt.Errorf("update XLM values: %w", err)
	}

	// Step 5: Calculate delegations
	s.logger.Info("calculating delegations")
	if err := s.calculateDelegations(ctx); err != nil {
		return result, fmt.Errorf("calculate delegations: %w", err)
	}

	// Step 6: Fetch association tags
	s.logger.Info("fetching association tags")
	if err := s.syncAssociationTags(ctx); err != nil {
		return result, fmt.Errorf("sync association tags: %w", err)
	}

	// Step 7: Calculate reputation scores
	s.logger.Info("calculating reputation scores")
	if err := s.calculateReputationScores(ctx); err != nil {
		// Log error but don't fail sync - reputation is non-critical
		s.logger.Error("failed to calculate reputation scores", "error", err)
	}

	// Get final stats
	stats, err := s.repo.GetSyncStats(ctx)
	if err != nil {
		return result, fmt.Errorf("get sync stats: %w", err)
	}
	result.Stats = stats

	s.logger.Info("sync completed",
		"accounts", stats.TotalAccounts,
		"persons", stats.TotalPersons,
		"companies", stats.TotalCompanies,
		"total_xlm_value", stats.TotalXLMValue,
	)

	return result, nil
}

// calculateReputationScores computes and stores weighted reputation scores for all accounts.
func (s *Syncer) calculateReputationScores(ctx context.Context) error {
	// Create reputation repository
	repRepo, err := reputation.NewRepository(s.repo.Pool())
	if err != nil {
		return fmt.Errorf("create reputation repository: %w", err)
	}

	// Fetch all rating edges (A/B/C/D relationships)
	ratings, err := repRepo.GetRatingEdges(ctx)
	if err != nil {
		return fmt.Errorf("get rating edges: %w", err)
	}
	if len(ratings) == 0 {
		s.logger.Info("no reputation ratings found")
		return nil
	}

	// Fetch portfolios
	portfolios, err := repRepo.GetPortfolios(ctx)
	if err != nil {
		return fmt.Errorf("get portfolios: %w", err)
	}

	// Fetch connection counts
	connections, err := repRepo.GetConnectionCounts(ctx)
	if err != nil {
		return fmt.Errorf("get connection counts: %w", err)
	}

	// Calculate scores
	calc := reputation.NewCalculator()
	scores := calc.CalculateScores(ratings, portfolios, connections)

	// Store scores
	if err := repRepo.UpsertScores(ctx, scores); err != nil {
		return fmt.Errorf("upsert reputation scores: %w", err)
	}

	s.logger.Info("calculated reputation scores", "count", len(scores))
	return nil
}
