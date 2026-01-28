package handler

import (
	"errors"
	"net/http"

	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/service"
	"github.com/mtlprog/lore/internal/template"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	stellar  *service.StellarService
	accounts *repository.AccountRepository
	tmpl     *template.Templates
}

// New creates a new Handler with the given dependencies.
// Returns error if any required dependency is nil.
func New(stellar *service.StellarService, accounts *repository.AccountRepository, tmpl *template.Templates) (*Handler, error) {
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
