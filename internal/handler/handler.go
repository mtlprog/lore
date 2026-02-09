//go:generate mockery

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
)

// StellarServicer defines the interface for Stellar blockchain operations.
type StellarServicer interface {
	GetAccountDetail(ctx context.Context, accountID string) (*model.AccountDetail, error)
	GetAccountOperations(ctx context.Context, accountID, cursor string, limit int) (*model.OperationsPage, error)
	GetTransactionDetail(ctx context.Context, txHash string) (*model.Transaction, error)
	GetTokenDetail(ctx context.Context, code, issuer string) (*model.TokenDetail, error)
	GetTokenOrderbook(ctx context.Context, code, issuer string, limit int) (*model.TokenOrderbook, error)
	GetIssuerNFTMetadata(ctx context.Context, issuerID, assetCode string) (*model.NFTMetadata, error)
	FetchStellarToml(ctx context.Context, homeDomain string) (*model.StellarTomlCurrency, string, error)
	GetRawAccountData(ctx context.Context, accountID string) (map[string]string, error)
	GetAccountSequence(ctx context.Context, accountID string) (int64, error)
}

// AccountQuerier defines the interface for account data access.
type AccountQuerier interface {
	GetStats(ctx context.Context) (*repository.Stats, error)
	GetPersons(ctx context.Context, limit int, offset int) ([]repository.PersonRow, error)
	GetCorporate(ctx context.Context, limit int, offset int) ([]repository.CorporateRow, error)
	GetSynthetic(ctx context.Context, limit int, offset int) ([]repository.SyntheticRow, error)
	GetRelationships(ctx context.Context, accountID string) ([]repository.RelationshipRow, error)
	GetTrustRatings(ctx context.Context, accountID string) (*repository.TrustRating, error)
	GetConfirmedRelationships(ctx context.Context, accountID string) (map[string]bool, error)
	GetAccountInfo(ctx context.Context, accountID string) (*repository.AccountInfo, error)
	GetAccountNames(ctx context.Context, accountIDs []string) (map[string]string, error)
	GetAllTags(ctx context.Context) ([]repository.TagRow, error)
	SearchAccounts(ctx context.Context, query string, tags []string, limit int, offset int, sortBy repository.SearchSortOrder) ([]repository.SearchAccountRow, error)
	CountSearchAccounts(ctx context.Context, query string, tags []string) (int, error)
	GetLPShares(ctx context.Context, accountID string) ([]repository.LPShareRow, error)
}

// ReputationQuerier defines the interface for reputation data access.
type ReputationQuerier interface {
	GetScore(ctx context.Context, accountID string) (*model.ReputationScore, error)
	GetGraph(ctx context.Context, accountID string) (*model.ReputationGraph, error)
}

// TemplateRenderer defines the interface for template rendering.
type TemplateRenderer interface {
	Render(w io.Writer, name string, data any) error
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	stellar    StellarServicer
	accounts   AccountQuerier
	reputation ReputationQuerier
	tmpl       TemplateRenderer
}

// New creates a new Handler with the given dependencies.
// Returns error if any required dependency is nil.
// reputation can be nil (feature is optional).
func New(stellar StellarServicer, accounts AccountQuerier, reputation ReputationQuerier, tmpl TemplateRenderer) (*Handler, error) {
	if stellar == nil {
		return nil, errors.New("stellar service is required")
	}
	if accounts == nil {
		return nil, errors.New("account repository is required")
	}
	if tmpl == nil {
		return nil, errors.New("templates are required")
	}
	return &Handler{
		stellar:    stellar,
		accounts:   accounts,
		reputation: reputation,
		tmpl:       tmpl,
	}, nil
}

// RegisterRoutes registers all HTTP routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /accounts/{id}", h.Account)
	mux.HandleFunc("GET /accounts/{id}/reputation", h.Reputation)
	mux.HandleFunc("GET /transactions/{hash}", h.Transaction)
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /tokens/{issuer}/{code}", h.Token)

	// Init form routes
	mux.HandleFunc("GET /init", h.InitLanding)
	mux.HandleFunc("GET /init/participant", h.InitParticipant)
	mux.HandleFunc("POST /init/participant", h.InitParticipantSubmit)
	mux.HandleFunc("GET /init/corporate", h.InitCorporate)
	mux.HandleFunc("POST /init/corporate", h.InitCorporateSubmit)
}

// RegisterStaticRoutes registers routes for static files (favicon, og-image, etc.).
// staticHandler should be created from static.Handler().
func RegisterStaticRoutes(mux *http.ServeMux, staticHandler http.Handler) {
	mux.Handle("GET /favicon.svg", staticHandler)
	mux.Handle("GET /og-image.svg", staticHandler)
	mux.Handle("GET /og-image.png", staticHandler)
	mux.Handle("GET /favicon-32x32.png", staticHandler)
	mux.Handle("GET /favicon-16x16.png", staticHandler)
	mux.Handle("GET /apple-touch-icon.png", staticHandler)
	mux.Handle("GET /skill.md", staticHandler)

	// Robots.txt for SEO
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\nSitemap: https://lore.mtlprog.xyz/sitemap.xml\n"))
	})
}
