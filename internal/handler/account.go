package handler

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/service"
)

// AccountData holds data for the account detail page template.
type AccountData struct {
	Account *model.AccountDetail
}

// Account handles the account detail page.
func (h *Handler) Account(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	if accountID == "" {
		http.Error(w, "Account ID required", http.StatusBadRequest)
		return
	}

	account, err := h.stellar.GetAccountDetail(ctx, accountID)
	if err != nil {
		if service.IsNotFound(err) {
			http.Error(w, "Account not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to fetch account", "account_id", accountID, "error", err)
		http.Error(w, "Failed to fetch account", http.StatusInternalServerError)
		return
	}

	data := AccountData{
		Account: account,
	}

	var buf bytes.Buffer
	if err := h.tmpl.Render(&buf, "account.html", data); err != nil {
		slog.Error("failed to render account template", "account_id", accountID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "account_id", accountID, "error", err)
	}
}
