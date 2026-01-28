package handler

import (
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/model"
)

// HomeData holds data for the home page template.
type HomeData struct {
	Persons   *model.AccountsPage
	Companies *model.AccountsPage
}

// Home handles the main page showing Persons and Companies.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	personsCursor := r.URL.Query().Get("persons_cursor")
	companiesCursor := r.URL.Query().Get("companies_cursor")

	persons, err := h.stellar.GetAccountsWithAsset(
		ctx,
		config.TokenMTLAP,
		config.TokenIssuer,
		personsCursor,
		config.DefaultPageLimit,
	)
	if err != nil {
		slog.Error("failed to fetch persons", "error", err)
		http.Error(w, "Failed to fetch persons", http.StatusInternalServerError)
		return
	}

	companies, err := h.stellar.GetAccountsWithAsset(
		ctx,
		config.TokenMTLAC,
		config.TokenIssuer,
		companiesCursor,
		config.DefaultPageLimit,
	)
	if err != nil {
		slog.Error("failed to fetch companies", "error", err)
		http.Error(w, "Failed to fetch companies", http.StatusInternalServerError)
		return
	}

	data := HomeData{
		Persons:   persons,
		Companies: companies,
	}

	if err := h.tmpl.Render(w, "home.html", data); err != nil {
		slog.Error("failed to render home template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
