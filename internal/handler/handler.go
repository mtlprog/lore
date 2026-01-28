package handler

import (
	"net/http"

	"github.com/mtlprog/lore/internal/service"
	"github.com/mtlprog/lore/internal/template"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	stellar *service.StellarService
	tmpl    *template.Templates
}

// New creates a new Handler with the given dependencies.
func New(stellar *service.StellarService, tmpl *template.Templates) *Handler {
	return &Handler{
		stellar: stellar,
		tmpl:    tmpl,
	}
}

// RegisterRoutes registers all HTTP routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /accounts/{id}", h.Account)
}
