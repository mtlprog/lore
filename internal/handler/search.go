package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/mtlprog/lore/internal/config"
)

const (
	minSearchQueryLength = 2
	maxSearchQueryLength = 100
)

// SearchData holds data for the search page template.
type SearchData struct {
	Query        string
	QueryTooLong bool
	Accounts     []SearchAccountDisplay
	TotalCount   int
	Offset       int
	NextOffset   int
	HasMore      bool
}

// SearchAccountDisplay represents an account for the search results template.
type SearchAccountDisplay struct {
	AccountID     string
	Name          string
	MTLAPBalance  float64
	MTLACBalance  float64
	TotalXLMValue float64
	IsPerson      bool
	IsCorporate   bool
}

// Search handles the search page.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse and validate search query
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Parse offset for pagination
	offsetParam := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset < 0 {
		if offsetParam != "" {
			slog.Debug("invalid offset parameter, defaulting to 0", "value", offsetParam)
		}
		offset = 0
	}

	var accounts []SearchAccountDisplay
	var totalCount int
	hasMore := false
	queryTooLong := false

	// Check if query exceeds maximum length
	if len(query) > maxSearchQueryLength {
		slog.Debug("search query exceeds maximum length", "length", len(query), "max", maxSearchQueryLength)
		queryTooLong = true
	} else if len(query) >= minSearchQueryLength {
		// Only search if query is valid
		// Fetch total count
		totalCount, err = h.accounts.CountSearchAccounts(ctx, query)
		if err != nil {
			slog.Error("failed to count search accounts", "query", query, "error", err)
			http.Error(w, "Failed to search accounts", http.StatusInternalServerError)
			return
		}

		// Fetch accounts with pagination (fetch one extra to check for more)
		rows, err := h.accounts.SearchAccounts(ctx, query, config.DefaultPageLimit+1, offset)
		if err != nil {
			slog.Error("failed to search accounts", "query", query, "offset", offset, "error", err)
			http.Error(w, "Failed to search accounts", http.StatusInternalServerError)
			return
		}

		hasMore = len(rows) > config.DefaultPageLimit
		if hasMore {
			rows = rows[:config.DefaultPageLimit]
		}

		// Convert to display structs
		for _, row := range rows {
			accounts = append(accounts, SearchAccountDisplay{
				AccountID:     row.AccountID,
				Name:          row.Name,
				MTLAPBalance:  row.MTLAPBalance,
				MTLACBalance:  row.MTLACBalance,
				TotalXLMValue: row.TotalXLMValue,
				IsPerson:      row.MTLAPBalance > 0 && row.MTLAPBalance <= 5,
				IsCorporate:   row.MTLACBalance > 0 && row.MTLACBalance <= 4,
			})
		}
	}

	data := SearchData{
		Query:        query,
		QueryTooLong: queryTooLong,
		Accounts:     accounts,
		TotalCount:   totalCount,
		Offset:       offset,
		NextOffset:   offset + config.DefaultPageLimit,
		HasMore:      hasMore,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "search.html", data); err != nil {
		slog.Error("failed to render search template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}
