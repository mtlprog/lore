package api

import (
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/bsn"
	"github.com/samber/lo"
)

// GetRelationships handles GET /api/v1/accounts/{id}/relationships.
//
//	@Summary		Get account relationships
//	@Description	Returns relationships grouped by category (family, work, network, ownership, social)
//	@Tags			relationships
//	@Produce		json
//	@Param			id			path		string	true	"Stellar account ID"
//	@Param			type		query		string	false	"Filter by relationship type (e.g. Spouse, Employer)"
//	@Param			confirmed	query		bool	false	"Filter to only confirmed relationships"
//	@Param			mutual		query		bool	false	"Filter to only mutual relationships"
//	@Success		200			{array}		RelationshipCategoryResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/api/v1/accounts/{id}/relationships [get]
func (h *Handler) GetRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.validateAccountID(w, r)
	if !ok {
		return
	}

	exists, err := h.accounts.AccountExists(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to check account existence", "account_id", accountID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to check account")
		return
	}
	if !exists {
		h.writeError(w, http.StatusNotFound, "account not found")
		return
	}

	filterType := r.URL.Query().Get("type")
	filterConfirmed := r.URL.Query().Get("confirmed") == "true"
	filterMutual := r.URL.Query().Get("mutual") == "true"

	relationships, err := h.accounts.GetRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch relationships", "account_id", accountID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to fetch relationships")
		return
	}

	confirmed, err := h.accounts.GetConfirmedRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch confirmed relationships", "account_id", accountID, "error", err)
		confirmed = make(map[string]bool)
	}

	categories := bsn.GroupRelationships(accountID, relationships, confirmed)
	resp := convertCategories(categories)

	// Apply optional filters to each category's relationships
	hasFilters := filterType != "" || filterConfirmed || filterMutual
	if hasFilters {
		for i := range resp {
			resp[i].Relationships = lo.Filter(resp[i].Relationships, func(rel RelationshipResponse, _ int) bool {
				if filterType != "" && rel.Type != filterType {
					return false
				}
				if filterConfirmed && !rel.IsConfirmed {
					return false
				}
				if filterMutual && !rel.IsMutual {
					return false
				}
				return true
			})
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}
