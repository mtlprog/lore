package handler

import (
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/service"
	"github.com/samber/lo"
)

// TransactionData holds data for the transaction detail page template.
type TransactionData struct {
	Transaction  *model.Transaction
	AccountNames map[string]string // Map of account ID to name for linked accounts
}

// Transaction handles the transaction detail page.
func (h *Handler) Transaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	txHash := r.PathValue("hash")

	if txHash == "" {
		http.Error(w, "Transaction hash required", http.StatusBadRequest)
		return
	}

	// Stellar transaction hashes are 64 hex characters
	if len(txHash) != 64 {
		http.Error(w, "Invalid transaction hash format", http.StatusBadRequest)
		return
	}

	tx, err := h.stellar.GetTransactionDetail(ctx, txHash)
	if err != nil {
		if service.IsNotFound(err) {
			http.Error(w, "Transaction not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to fetch transaction", "tx_hash", txHash, "error", err)
		http.Error(w, "Failed to fetch transaction", http.StatusInternalServerError)
		return
	}

	// Filter out claimable balance operations (spam)
	tx.Operations = lo.Filter(tx.Operations, func(op model.Operation, _ int) bool {
		return op.Type != "create_claimable_balance" && op.Type != "claim_claimable_balance"
	})
	tx.OperationCount = len(tx.Operations)

	// Collect account IDs and look up names
	accountIDs := collectAccountIDs(tx.Operations)
	// Add source account
	accountIDs = append(accountIDs, tx.SourceAccount)
	accountIDs = lo.Uniq(accountIDs)

	var accountNames map[string]string
	if len(accountIDs) > 0 {
		accountNames, err = h.accounts.GetAccountNames(ctx, accountIDs)
		if err != nil {
			slog.Error("failed to fetch account names", "tx_hash", txHash, "lookup_count", len(accountIDs), "error", err)
			accountNames = make(map[string]string)
		}
	}

	data := TransactionData{
		Transaction:  tx,
		AccountNames: accountNames,
	}

	buf := h.getBuffer()
	defer h.putBuffer(buf)

	if err := h.tmpl.Render(buf, "transaction.html", data); err != nil {
		slog.Error("failed to render transaction template", "tx_hash", txHash, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		slog.Debug("failed to write response", "tx_hash", txHash, "error", err)
	}
}
