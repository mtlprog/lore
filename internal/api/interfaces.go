package api

import (
	"context"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
)

// accountQuerierBase defines the interface for account data access needed by the API.
type accountQuerierBase interface {
	GetStats(ctx context.Context) (*repository.Stats, error)
	GetPersons(ctx context.Context, limit int, offset int) ([]repository.PersonRow, error)
	GetCorporate(ctx context.Context, limit int, offset int) ([]repository.CorporateRow, error)
	GetSynthetic(ctx context.Context, limit int, offset int) ([]repository.SyntheticRow, error)
	GetRelationships(ctx context.Context, accountID string) ([]repository.RelationshipRow, error)
	GetTrustRatings(ctx context.Context, accountID string) (*repository.TrustRating, error)
	GetConfirmedRelationships(ctx context.Context, accountID string) (map[string]bool, error)
	GetAccountInfo(ctx context.Context, accountID string) (*repository.AccountInfo, error)
	GetAccountNames(ctx context.Context, accountIDs []string) (map[string]string, error)
	SearchAccounts(ctx context.Context, query string, tags []string, limit int, offset int, sortBy repository.SearchSortOrder) ([]repository.SearchAccountRow, error)
	CountSearchAccounts(ctx context.Context, query string, tags []string) (int, error)
	GetLPShares(ctx context.Context, accountID string) ([]repository.LPShareRow, error)
	CountPersons(ctx context.Context) (int, error)
	CountCorporate(ctx context.Context) (int, error)
	CountSynthetic(ctx context.Context) (int, error)
	GetAccountMetadata(ctx context.Context, accountID string) (*repository.AccountMetadata, error)
}

// reputationQuerierBase defines the interface for reputation data access needed by the API.
type reputationQuerierBase interface {
	GetScore(ctx context.Context, accountID string) (*model.ReputationScore, error)
	GetGraph(ctx context.Context, accountID string) (*model.ReputationGraph, error)
}
