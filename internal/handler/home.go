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
	Companies           []repository.CompanyRow
	PersonsOffset       int
	CompaniesOffset     int
	NextPersonsOffset   int
	NextCompaniesOffset int
	HasMorePersons      bool
	HasMoreCompanies    bool
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
	companiesOffsetParam := r.URL.Query().Get("companies_offset")
	companiesOffset, err := strconv.Atoi(companiesOffsetParam)
	if err != nil || companiesOffset < 0 {
		if companiesOffsetParam != "" {
			slog.Debug("invalid companies_offset parameter, defaulting to 0", "value", companiesOffsetParam)
		}
		companiesOffset = 0
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

	companies, err := h.accounts.GetCompanies(ctx, config.DefaultPageLimit+1, companiesOffset)
	if err != nil {
		slog.Error("failed to fetch companies", "offset", companiesOffset, "error", err)
		http.Error(w, "Failed to fetch companies", http.StatusInternalServerError)
		return
	}

	hasMorePersons := len(persons) > config.DefaultPageLimit
	hasMoreCompanies := len(companies) > config.DefaultPageLimit

	if hasMorePersons {
		persons = persons[:config.DefaultPageLimit]
	}
	if hasMoreCompanies {
		companies = companies[:config.DefaultPageLimit]
	}

	data := HomeData{
		Stats:               stats,
		Persons:             persons,
		Companies:           companies,
		PersonsOffset:       personsOffset,
		CompaniesOffset:     companiesOffset,
		NextPersonsOffset:   personsOffset + config.DefaultPageLimit,
		NextCompaniesOffset: companiesOffset + config.DefaultPageLimit,
		HasMorePersons:      hasMorePersons,
		HasMoreCompanies:    hasMoreCompanies,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "home.html", data); err != nil {
		slog.Error("failed to render home template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
