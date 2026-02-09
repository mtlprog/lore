package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/mtlprog/lore/internal/bsn"
	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/service"
	"github.com/samber/lo"
)

// AccountData holds data for the account detail page template.
type AccountData struct {
	Account         *model.AccountDetail
	Operations      *model.OperationsPage
	AccountNames    map[string]string      // Map of account ID to name for linked accounts
	ReputationScore *model.ReputationScore // Weighted reputation score (optional)
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

	// Fetch relationships from database
	relationships, err := h.accounts.GetRelationships(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch relationships", "account_id", accountID, "error", err)
		// Continue without relationships - don't fail the whole page
		relationships = nil
	}

	// Fetch trust ratings
	trustRating, err := h.accounts.GetTrustRatings(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch trust ratings", "account_id", accountID, "error", err)
		trustRating = nil
	}

	// Fetch confirmed relationships
	confirmed, err := h.accounts.GetConfirmedRelationships(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch confirmed relationships", "account_id", accountID, "error", err)
		confirmed = make(map[string]bool)
	}

	// Fetch account info from database (for XLM valuation)
	accountInfo, err := h.accounts.GetAccountInfo(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch account info", "account_id", accountID, "error", err)
		accountInfo = nil
	}

	// Fetch operations with cursor-based pagination
	opsCursor := r.URL.Query().Get("ops_cursor")
	const operationsLimit = 10
	operations, err := h.stellar.GetAccountOperations(ctx, accountID, opsCursor, operationsLimit)
	if err != nil {
		slog.Error("failed to fetch operations", "account_id", accountID, "error", err)
		operations = nil
	}

	// Collect account IDs from operations for name lookup
	var accountNames map[string]string
	if operations != nil && len(operations.Operations) > 0 {
		accountIDs := collectAccountIDs(operations.Operations)
		if len(accountIDs) > 0 {
			accountNames, err = h.accounts.GetAccountNames(ctx, accountIDs)
			if err != nil {
				slog.Error("failed to fetch account names", "account_id", accountID, "lookup_count", len(accountIDs), "error", err)
				accountNames = make(map[string]string)
			}
		}
	}

	// Fetch LP shares from database
	// Continue without LP shares on error - don't fail the whole page for non-critical data
	lpRows, err := h.accounts.GetLPShares(ctx, accountID)
	if err != nil {
		slog.Error("failed to fetch LP shares", "account_id", accountID, "error", err)
		lpRows = nil
	}

	// Convert LP rows to display model
	account.LPShares = convertLPShares(lpRows)

	// Separate NFTs from regular tokens (NFTs have balance == "0.0000001")
	// FilterReject returns (kept, rejected) - we keep regular tokens, reject NFTs
	tokens, nfts := lo.FilterReject(account.Trustlines, func(t model.Trustline, _ int) bool {
		return t.AssetIssuer == "native" || t.Balance != "0.0000001"
	})
	account.Trustlines = tokens
	account.NFTTrustlines = nfts

	// Sort trustlines: XLM first, then by balance (already sorted by balance from service)
	sort.SliceStable(account.Trustlines, func(i, j int) bool {
		if account.Trustlines[i].AssetIssuer == "native" {
			return true
		}
		if account.Trustlines[j].AssetIssuer == "native" {
			return false
		}
		return false // Keep existing order (by balance) for non-native assets
	})

	// Process relationships into categories
	account.Categories = bsn.GroupRelationships(accountID, relationships, confirmed)

	// Set XLM valuation for corporate accounts
	if accountInfo != nil && accountInfo.MTLACBalance > 0 {
		account.IsCorporate = true
		account.TotalXLMValue = accountInfo.TotalXLMValue
	}

	// Process trust rating
	if trustRating != nil && trustRating.Total > 0 {
		account.TrustRating = &model.TrustRating{
			CountA:  trustRating.CountA,
			CountB:  trustRating.CountB,
			CountC:  trustRating.CountC,
			CountD:  trustRating.CountD,
			Total:   trustRating.Total,
			Score:   trustRating.Score,
			Grade:   bsn.CalculateTrustGrade(trustRating.Score),
			Percent: int((trustRating.Score / 4.0) * 100),
		}
	}

	// Fetch weighted reputation score (optional feature)
	var reputationScore *model.ReputationScore
	if h.reputation != nil {
		reputationScore, err = h.reputation.GetScore(ctx, accountID)
		if err != nil {
			slog.Warn("failed to fetch reputation score, continuing without", "account_id", accountID, "error", err)
		}
	}

	data := AccountData{
		Account:         account,
		Operations:      operations,
		AccountNames:    accountNames,
		ReputationScore: reputationScore,
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

// collectAccountIDs extracts all unique account IDs from operations.
// This includes From, To, SourceAccount, and DataValue if it looks like a Stellar account ID.
func collectAccountIDs(ops []model.Operation) []string {
	idSet := make(map[string]struct{})

	for _, op := range ops {
		if op.From != "" {
			idSet[op.From] = struct{}{}
		}
		if op.To != "" {
			idSet[op.To] = struct{}{}
		}
		if op.SourceAccount != "" {
			idSet[op.SourceAccount] = struct{}{}
		}
		// Check if DataValue looks like a Stellar account ID (starts with G, 56 chars)
		if op.DataValue != "" && isStellarAccountID(op.DataValue) {
			idSet[op.DataValue] = struct{}{}
		}
	}

	return lo.Keys(idSet)
}

// isStellarAccountID checks if a string looks like a Stellar account ID.
func isStellarAccountID(s string) bool {
	return len(s) == 56 && (s[0] == 'G' || s[0] == 'M')
}

// convertLPShares converts LP share rows to display models.
func convertLPShares(rows []repository.LPShareRow) []model.LPShareDisplay {
	if len(rows) == 0 {
		return nil
	}

	return lo.Map(rows, func(r repository.LPShareRow, _ int) model.LPShareDisplay {
		// Calculate share percentage
		sharePercent := "0%"
		if r.TotalShares > 0 {
			pct := (r.ShareBalance / r.TotalShares) * 100
			if pct < 0.01 {
				sharePercent = "<0.01%"
			} else {
				sharePercent = fmt.Sprintf("%.2f%%", pct)
			}
		}

		// Calculate proportional reserves
		shareRatio := float64(0)
		if r.TotalShares > 0 {
			shareRatio = r.ShareBalance / r.TotalShares
		}

		return model.LPShareDisplay{
			PoolID:       r.PoolID,
			ShareBalance: fmt.Sprintf("%.7f", r.ShareBalance),
			SharePercent: sharePercent,
			ReserveA: model.LPReserveDisplay{
				AssetCode:   r.ReserveACode,
				AssetIssuer: r.ReserveAIssuer,
				Amount:      fmt.Sprintf("%.4f", r.ReserveAAmount*shareRatio),
			},
			ReserveB: model.LPReserveDisplay{
				AssetCode:   r.ReserveBCode,
				AssetIssuer: r.ReserveBIssuer,
				Amount:      fmt.Sprintf("%.4f", r.ReserveBAmount*shareRatio),
			},
			XLMValue: r.XLMValue,
		}
	})
}
