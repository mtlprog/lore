package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/repository"
)

// HomeData holds data for the home page template.
type HomeData struct {
	Stats               *repository.Stats
	Persons             []repository.PersonRow
	Corporate           []repository.CorporateRow
	PersonsOffset       int
	CorporateOffset     int
	NextPersonsOffset   int
	NextCorporateOffset int
	HasMorePersons      bool
	HasMoreCorporate    bool
}

// Home handles the main page showing Persons and Companies.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	personsOffsetParam := r.URL.Query().Get("persons_offset")
	personsOffset, err := strconv.Atoi(personsOffsetParam)
	if err != nil || personsOffset < 0 {
		if personsOffsetParam != "" {
			slog.Debug("invalid persons_offset parameter, defaulting to 0", "value", personsOffsetParam)
		}
		personsOffset = 0
	}
	corporateOffsetParam := r.URL.Query().Get("corporate_offset")
	corporateOffset, err := strconv.Atoi(corporateOffsetParam)
	if err != nil || corporateOffset < 0 {
		if corporateOffsetParam != "" {
			slog.Debug("invalid corporate_offset parameter, defaulting to 0", "value", corporateOffsetParam)
		}
		corporateOffset = 0
	}

	stats, err := h.accounts.GetStats(ctx)
	if err != nil {
		slog.Error("failed to fetch stats", "error", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	persons, err := h.accounts.GetPersons(ctx, config.DefaultPageLimit+1, personsOffset)
	if err != nil {
		slog.Error("failed to fetch persons", "offset", personsOffset, "error", err)
		http.Error(w, "Failed to fetch persons", http.StatusInternalServerError)
		return
	}

	corporate, err := h.accounts.GetCorporate(ctx, config.DefaultPageLimit+1, corporateOffset)
	if err != nil {
		slog.Error("failed to fetch corporate", "offset", corporateOffset, "error", err)
		http.Error(w, "Failed to fetch corporate accounts", http.StatusInternalServerError)
		return
	}

	hasMorePersons := len(persons) > config.DefaultPageLimit
	hasMoreCorporate := len(corporate) > config.DefaultPageLimit

	if hasMorePersons {
		persons = persons[:config.DefaultPageLimit]
	}
	if hasMoreCorporate {
		corporate = corporate[:config.DefaultPageLimit]
	}

	data := HomeData{
		Stats:               stats,
		Persons:             persons,
		Corporate:           corporate,
		PersonsOffset:       personsOffset,
		CorporateOffset:     corporateOffset,
		NextPersonsOffset:   personsOffset + config.DefaultPageLimit,
		NextCorporateOffset: corporateOffset + config.DefaultPageLimit,
		HasMorePersons:      hasMorePersons,
		HasMoreCorporate:    hasMoreCorporate,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "home.html", data); err != nil {
		slog.Error("failed to render home template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}
