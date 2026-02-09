package api

import (
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/bsn"
	"github.com/mtlprog/lore/internal/model"
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
	accountID := r.PathValue("id")

	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account ID is required")
		return
	}

	if !isValidStellarID(accountID) {
		writeError(w, http.StatusBadRequest, "invalid Stellar account ID format")
		return
	}

	filterType := r.URL.Query().Get("type")
	filterConfirmed := r.URL.Query().Get("confirmed") == "true"
	filterMutual := r.URL.Query().Get("mutual") == "true"

	relationships, err := h.accounts.GetRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch relationships", "account_id", accountID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch relationships")
		return
	}

	confirmed, err := h.accounts.GetConfirmedRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch confirmed relationships", "account_id", accountID, "error", err)
		confirmed = make(map[string]bool)
	}

	categories := bsn.GroupRelationships(accountID, relationships, confirmed)

	resp := lo.Map(categories, func(cat model.RelationshipCategory, _ int) RelationshipCategoryResponse {
		rels := lo.Map(cat.Relationships, func(rel model.Relationship, _ int) RelationshipResponse {
			return RelationshipResponse{
				Type:        rel.Type,
				TargetID:    rel.TargetID,
				TargetName:  rel.TargetName,
				Direction:   rel.Direction,
				IsMutual:    rel.IsMutual,
				IsConfirmed: rel.IsConfirmed,
			}
		})

		// Apply filters
		if filterType != "" {
			rels = lo.Filter(rels, func(r RelationshipResponse, _ int) bool {
				return r.Type == filterType
			})
		}
		if filterConfirmed {
			rels = lo.Filter(rels, func(r RelationshipResponse, _ int) bool {
				return r.IsConfirmed
			})
		}
		if filterMutual {
			rels = lo.Filter(rels, func(r RelationshipResponse, _ int) bool {
				return r.IsMutual
			})
		}

		return RelationshipCategoryResponse{
			Name:          cat.Name,
			Color:         cat.Color,
			Relationships: rels,
		}
	})

	writeJSON(w, http.StatusOK, resp)
}
