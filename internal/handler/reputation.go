package handler

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
)

// ReputationData holds data for the reputation page template.
type ReputationData struct {
	AccountID   string
	AccountName string
	Score       *model.ReputationScore
	Graph       *model.ReputationGraph
}

// Reputation handles GET /accounts/{id}/reputation.
func (h *Handler) Reputation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	if accountID == "" {
		http.NotFound(w, r)
		return
	}

	// Check if reputation service is available
	if h.reputation == nil {
		slog.Debug("reputation service not available")
		http.Error(w, "Reputation feature not available", http.StatusServiceUnavailable)
		return
	}

	// Fetch reputation graph (includes score)
	graph, err := h.reputation.GetGraph(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch reputation graph", "account_id", accountID, "error", err)
		http.Error(w, "Failed to load reputation data", http.StatusInternalServerError)
		return
	}

	var score *model.ReputationScore
	if graph != nil {
		score = graph.Score
	}

	// If no graph was returned, try to at least get the score
	if graph == nil {
		score, err = h.reputation.GetScore(ctx, accountID)
		if err != nil {
			slog.Error("failed to fetch reputation score", "account_id", accountID, "error", err)
		}
	}

	// Get account name for display
	accountName := accountID
	names, err := h.accounts.GetAccountNames(ctx, []string{accountID})
	if err == nil && names[accountID] != "" {
		accountName = names[accountID]
	}

	data := ReputationData{
		AccountID:   accountID,
		AccountName: accountName,
		Score:       score,
		Graph:       graph,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "reputation.html", data); err != nil {
		slog.Error("failed to render reputation template", "account_id", accountID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "account_id", accountID, "error", err)
	}
}
