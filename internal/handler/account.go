package handler

import (
	"log"
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
		log.Printf("Error fetching account %s: %v", accountID, err)
		http.Error(w, "Failed to fetch account", http.StatusInternalServerError)
		return
	}

	data := AccountData{
		Account: account,
	}

	if err := h.tmpl.Render(w, "account.html", data); err != nil {
		log.Printf("Error rendering account template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
