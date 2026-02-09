package api

import (
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
	"github.com/samber/lo"
)

// GetReputation handles GET /api/v1/accounts/{id}/reputation.
//
//	@Summary		Get reputation graph
//	@Description	Returns the reputation graph including direct raters and raters of raters
//	@Tags			reputation
//	@Produce		json
//	@Param			id	path		string	true	"Stellar account ID"
//	@Success		200	{object}	ReputationGraphResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Router			/api/v1/accounts/{id}/reputation [get]
func (h *Handler) GetReputation(w http.ResponseWriter, r *http.Request) {
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

	if h.reputation == nil {
		writeError(w, http.StatusServiceUnavailable, "reputation feature not available")
		return
	}

	graph, err := h.reputation.GetGraph(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch reputation graph", "account_id", accountID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch reputation data")
		return
	}

	if graph == nil {
		// Try to get at least the score
		score, err := h.reputation.GetScore(ctx, accountID)
		if err != nil {
			slog.Error("api: failed to fetch reputation score", "account_id", accountID, "error", err)
		}

		resp := ReputationGraphResponse{
			TargetAccountID: accountID,
			TargetName:      accountID,
			Score:           convertReputationScore(score),
			Level1Nodes:     []ReputationNodeResponse{},
			Level2Nodes:     []ReputationNodeResponse{},
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	resp := ReputationGraphResponse{
		TargetAccountID: graph.TargetAccountID,
		TargetName:      graph.TargetName,
		Score:           convertReputationScore(graph.Score),
		Level1Nodes: lo.Map(graph.Level1Nodes, func(n model.ReputationNode, _ int) ReputationNodeResponse {
			return convertReputationNode(n)
		}),
		Level2Nodes: lo.Map(graph.Level2Nodes, func(n model.ReputationNode, _ int) ReputationNodeResponse {
			return convertReputationNode(n)
		}),
	}

	writeJSON(w, http.StatusOK, resp)
}

func convertReputationNode(n model.ReputationNode) ReputationNodeResponse {
	return ReputationNodeResponse{
		AccountID:    n.AccountID,
		Name:         n.Name,
		Rating:       n.Rating,
		Weight:       n.Weight,
		PortfolioXLM: n.PortfolioXLM,
		Connections:  n.Connections,
		OwnScore:     n.OwnScore,
		Distance:     n.Distance,
	}
}
