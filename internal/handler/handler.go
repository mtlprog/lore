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
}

// AccountQuerier defines the interface for account data access.
type AccountQuerier interface {
	GetStats(ctx context.Context) (*repository.Stats, error)
	GetPersons(ctx context.Context, limit int, offset int) ([]repository.PersonRow, error)
	GetCorporate(ctx context.Context, limit int, offset int) ([]repository.CorporateRow, error)
}

// TemplateRenderer defines the interface for template rendering.
type TemplateRenderer interface {
	Render(w io.Writer, name string, data any) error
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	stellar  StellarServicer
	accounts AccountQuerier
	tmpl     TemplateRenderer
}

// New creates a new Handler with the given dependencies.
// Returns error if any required dependency is nil.
func New(stellar StellarServicer, accounts AccountQuerier, tmpl TemplateRenderer) (*Handler, error) {
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
		stellar:  stellar,
		accounts: accounts,
		tmpl:     tmpl,
	}, nil
}

// RegisterRoutes registers all HTTP routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /accounts/{id}", h.Account)
}
