package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	accounts   accountQuerierBase
	reputation reputationQuerierBase
	bufferPool *sync.Pool // Pool of bytes.Buffer for JSON encoding
}

// New creates a new API Handler.
// reputation can be nil (feature is optional).
func New(accounts accountQuerierBase, reputation reputationQuerierBase) (*Handler, error) {
	if accounts == nil {
		return nil, errors.New("account repository is required")
	}
	return &Handler{
		accounts:   accounts,
		reputation: reputation,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}, nil
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/stats", h.Stats)
	mux.HandleFunc("GET /api/v1/accounts", h.ListAccounts)
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetAccount)
	mux.HandleFunc("GET /api/v1/accounts/{id}/reputation", h.GetReputation)
	mux.HandleFunc("GET /api/v1/accounts/{id}/relationships", h.GetRelationships)
	mux.HandleFunc("GET /api/v1/search", h.Search)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	buf := h.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		h.bufferPool.Put(buf)
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
		http.Error(w, `{"error":"internal server error","code":500}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, ErrorResponse{
		Error: msg,
		Code:  status,
	})
}

func isValidStellarID(id string) bool {
	return len(id) == 56 && id[0] == 'G'
}

// validateAccountID validates the account ID path parameter and writes an error response if invalid.
// Returns the account ID and true if valid, or empty string and false if invalid (error already written).
func (h *Handler) validateAccountID(w http.ResponseWriter, r *http.Request) (string, bool) {
	accountID := r.PathValue("id")
	if accountID == "" {
		h.writeError(w, http.StatusBadRequest, "account ID is required")
		return "", false
	}
	if !isValidStellarID(accountID) {
		h.writeError(w, http.StatusBadRequest, "invalid Stellar account ID format")
		return "", false
	}
	return accountID, true
}

// inferAccountType determines account type from token balances.
func inferAccountType(mtlapBalance, mtlacBalance, mtlaxBalance float64) string {
	if mtlacBalance > 0 && mtlacBalance <= 4 {
		return "corporate"
	}
	if mtlaxBalance > 0 && mtlapBalance == 0 {
		return "synthetic"
	}
	return "person"
}

func parseIntParam(r *http.Request, name string, defaultVal, maxVal int) int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	if maxVal > 0 && v > maxVal {
		return maxVal
	}
	return v
}
