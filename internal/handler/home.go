package handler

import (
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

	personsOffset, _ := strconv.Atoi(r.URL.Query().Get("persons_offset"))
	companiesOffset, _ := strconv.Atoi(r.URL.Query().Get("companies_offset"))

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

	if err := h.tmpl.Render(w, "home.html", data); err != nil {
		slog.Error("failed to render home template", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
