package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/reputation"
	"github.com/samber/lo"
)

// Search handles GET /api/v1/search.
//
//	@Summary		Search accounts
//	@Description	Search accounts by name or account ID, optionally filtered by tags
//	@Tags			search
//	@Produce		json
//	@Param			q		query		string	false	"Search query (min 2 chars)"
//	@Param			tags	query		string	false	"Comma-separated tag names"
//	@Param			sort	query		string	false	"Sort order"	Enums(balance, reputation)	default(balance)
//	@Param			limit	query		int		false	"Number of results"	default(20)	maximum(100)
//	@Param			offset	query		int		false	"Offset for pagination"	default(0)
//	@Success		200		{object}	PaginatedResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/search [get]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseIntParam(r, "limit", defaultLimit, maxLimit)
	offset := parseIntParam(r, "offset", 0, 0)

	// Parse tags from comma-separated string
	var tags []string
	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		tags = lo.FilterMap(strings.Split(tagsParam, ","), func(t string, _ int) (string, bool) {
			trimmed := strings.TrimSpace(t)
			return trimmed, trimmed != ""
		})
	}

	// Parse sort
	sortParam := r.URL.Query().Get("sort")
	repoSort := repository.SearchSortByBalance
	if sortParam == "reputation" {
		repoSort = repository.SearchSortByReputation
	}

	// Validate query length
	if len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search query too long (max 100 characters)")
		return
	}

	// Validate tag lengths
	tags = lo.Filter(tags, func(t string, _ int) bool { return len(t) <= 100 })

	// Discard too-short queries (still allow tag-only search)
	if len(query) < 2 {
		query = ""
	}

	// If no query and no tags, return empty result
	if query == "" && len(tags) == 0 {
		writeJSON(w, http.StatusOK, PaginatedResponse{
			Data: []AccountListItem{},
			Pagination: Pagination{
				Limit:  limit,
				Offset: offset,
				Total:  0,
			},
		})
		return
	}

	totalCount, err := h.accounts.CountSearchAccounts(ctx, query, tags)
	if err != nil {
		slog.Error("api: failed to count search results", "query", query, "tags", tags, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to search accounts")
		return
	}

	rows, err := h.accounts.SearchAccounts(ctx, query, tags, limit, offset, repoSort)
	if err != nil {
		slog.Error("api: failed to search accounts", "query", query, "tags", tags, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to search accounts")
		return
	}

	items := lo.Map(rows, func(row repository.SearchAccountRow, _ int) AccountListItem {
		grade := ""
		if row.ReputationScore > 0 {
			grade = reputation.ScoreToGrade(row.ReputationScore)
		}

		return AccountListItem{
			ID:              row.AccountID,
			Name:            row.Name,
			Type:            inferAccountType(row.MTLAPBalance, row.MTLACBalance, row.MTLAXBalance),
			MTLAPBalance:    row.MTLAPBalance,
			MTLACBalance:    row.MTLACBalance,
			MTLAXBalance:    row.MTLAXBalance,
			TotalXLMValue:   row.TotalXLMValue,
			ReputationScore: row.ReputationScore,
			ReputationGrade: grade,
		}
	})

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data: items,
		Pagination: Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  totalCount,
		},
	})
}
