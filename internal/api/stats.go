package api

import (
	"log/slog"
	"net/http"
)

// Stats handles GET /api/v1/stats.
//
//	@Summary		Get aggregate statistics
//	@Description	Returns aggregate statistics for accounts, persons, companies, and synthetic tokens
//	@Tags			stats
//	@Produce		json
//	@Success		200	{object}	StatsResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/stats [get]
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.accounts.GetStats(r.Context())
	if err != nil {
		slog.Error("api: failed to fetch stats", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}

	h.writeJSON(w, http.StatusOK, StatsResponse{
		TotalAccounts:  stats.TotalAccounts,
		TotalPersons:   stats.TotalPersons,
		TotalCompanies: stats.TotalCompanies,
		TotalSynthetic: stats.TotalSynthetic,
		TotalXLMValue:  stats.TotalXLMValue,
	})
}
