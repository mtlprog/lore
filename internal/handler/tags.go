package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/repository"
)

// TagsData holds data for the tags page template.
type TagsData struct {
	AllTags      []repository.TagRow
	SelectedTags []string
	Accounts     []TaggedAccountDisplay
	TotalCount   int
	Offset       int
	NextOffset   int
	HasMore      bool
}

// TaggedAccountDisplay represents an account for the tags page template.
type TaggedAccountDisplay struct {
	AccountID     string
	Name          string
	About         string
	MTLAPBalance  float64
	MTLACBalance  float64
	TotalXLMValue float64
	IsPerson      bool
	IsCorporate   bool
}

// Tags handles the tags browsing page.
func (h *Handler) Tags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse selected tags from query params
	selectedTags := r.URL.Query()["tag"]

	// Parse offset for pagination
	offsetParam := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset < 0 {
		if offsetParam != "" {
			slog.Debug("invalid offset parameter, defaulting to 0", "value", offsetParam)
		}
		offset = 0
	}

	// Fetch all available tags
	allTags, err := h.accounts.GetAllTags(ctx)
	if err != nil {
		slog.Error("failed to fetch tags", "error", err)
		http.Error(w, "Failed to fetch tags", http.StatusInternalServerError)
		return
	}

	var accounts []TaggedAccountDisplay
	var totalCount int
	hasMore := false

	// Only fetch accounts if tags are selected
	if len(selectedTags) > 0 {
		// Fetch total count
		totalCount, err = h.accounts.CountAccountsByTags(ctx, selectedTags)
		if err != nil {
			slog.Error("failed to count accounts by tags", "tags", selectedTags, "error", err)
			http.Error(w, "Failed to count accounts", http.StatusInternalServerError)
			return
		}

		// Fetch accounts with pagination (fetch one extra to check for more)
		rows, err := h.accounts.GetAccountsByTags(ctx, selectedTags, config.DefaultPageLimit+1, offset)
		if err != nil {
			slog.Error("failed to fetch accounts by tags", "tags", selectedTags, "offset", offset, "error", err)
			http.Error(w, "Failed to fetch accounts", http.StatusInternalServerError)
			return
		}

		hasMore = len(rows) > config.DefaultPageLimit
		if hasMore {
			rows = rows[:config.DefaultPageLimit]
		}

		// Convert to display structs
		for _, row := range rows {
			accounts = append(accounts, TaggedAccountDisplay{
				AccountID:     row.AccountID,
				Name:          row.Name,
				About:         row.About,
				MTLAPBalance:  row.MTLAPBalance,
				MTLACBalance:  row.MTLACBalance,
				TotalXLMValue: row.TotalXLMValue,
				IsPerson:      row.MTLAPBalance > 0 && row.MTLAPBalance <= 5,
				IsCorporate:   row.MTLACBalance > 0 && row.MTLACBalance <= 4,
			})
		}
	}

	data := TagsData{
		AllTags:      allTags,
		SelectedTags: selectedTags,
		Accounts:     accounts,
		TotalCount:   totalCount,
		Offset:       offset,
		NextOffset:   offset + config.DefaultPageLimit,
		HasMore:      hasMore,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "tags.html", data); err != nil {
		slog.Error("failed to render tags template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}
