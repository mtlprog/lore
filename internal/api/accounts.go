package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mtlprog/lore/internal/bsn"
	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/reputation"
	"github.com/samber/lo"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListAccounts handles GET /api/v1/accounts.
//
//	@Summary		List accounts
//	@Description	Returns a paginated list of accounts, optionally filtered by type
//	@Tags			accounts
//	@Produce		json
//	@Param			type	query		string	false	"Account type filter"	Enums(person, corporate, synthetic)
//	@Param			limit	query		int		false	"Number of results"		default(20)	maximum(100)
//	@Param			offset	query		int		false	"Offset for pagination"	default(0)
//	@Success		200		{object}	PaginatedResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/accounts [get]
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountType := r.URL.Query().Get("type")
	limit := parseIntParam(r, "limit", defaultLimit, maxLimit)
	offset := parseIntParam(r, "offset", 0, 0)

	var items []AccountListItem
	var total int
	var err error

	switch accountType {
	case "person":
		items, total, err = h.listPersons(ctx, limit, offset)
	case "corporate":
		items, total, err = h.listCorporate(ctx, limit, offset)
	case "synthetic":
		items, total, err = h.listSynthetic(ctx, limit, offset)
	case "":
		items, total, err = h.listAll(ctx, limit, offset)
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid type: %s (valid: person, corporate, synthetic)", accountType))
		return
	}

	if err != nil {
		slog.Error("api: failed to list accounts", "type", accountType, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Data: items,
		Pagination: Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}

func (h *Handler) listPersons(ctx context.Context, limit, offset int) ([]AccountListItem, int, error) {
	persons, err := h.accounts.GetPersons(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get persons: %w", err)
	}

	total, err := h.accounts.CountPersons(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count persons: %w", err)
	}

	items := lo.Map(persons, func(p repository.PersonRow, _ int) AccountListItem {
		return AccountListItem{
			ID:             p.AccountID,
			Name:           p.Name,
			Type:           "person",
			MTLAPBalance:   p.MTLAPBalance,
			IsCouncilReady: p.IsCouncilReady,
			ReceivedVotes:  p.ReceivedVotes,
		}
	})

	return items, total, nil
}

func (h *Handler) listCorporate(ctx context.Context, limit, offset int) ([]AccountListItem, int, error) {
	corporate, err := h.accounts.GetCorporate(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get corporate: %w", err)
	}

	total, err := h.accounts.CountCorporate(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count corporate: %w", err)
	}

	items := lo.Map(corporate, func(c repository.CorporateRow, _ int) AccountListItem {
		return AccountListItem{
			ID:            c.AccountID,
			Name:          c.Name,
			Type:          "corporate",
			MTLACBalance:  c.MTLACBalance,
			TotalXLMValue: c.TotalXLMValue,
		}
	})

	return items, total, nil
}

func (h *Handler) listSynthetic(ctx context.Context, limit, offset int) ([]AccountListItem, int, error) {
	synthetic, err := h.accounts.GetSynthetic(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get synthetic: %w", err)
	}

	total, err := h.accounts.CountSynthetic(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count synthetic: %w", err)
	}

	items := lo.Map(synthetic, func(s repository.SyntheticRow, _ int) AccountListItem {
		grade := ""
		if s.ReputationScore > 0 {
			grade = reputation.ScoreToGrade(s.ReputationScore)
		}
		return AccountListItem{
			ID:              s.AccountID,
			Name:            s.Name,
			Type:            "synthetic",
			MTLAXBalance:    s.MTLAXBalance,
			ReputationScore: s.ReputationScore,
			ReputationGrade: grade,
		}
	})

	return items, total, nil
}

func (h *Handler) listAll(ctx context.Context, limit, offset int) ([]AccountListItem, int, error) {
	stats, err := h.accounts.GetStats(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("get stats: %w", err)
	}
	total := stats.TotalAccounts

	rows, err := h.accounts.GetAllAccounts(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get all accounts: %w", err)
	}

	items := lo.Map(rows, func(a repository.AllAccountRow, _ int) AccountListItem {
		accountType := "person"
		if a.MTLACBalance > 0 && a.MTLACBalance <= 4 {
			accountType = "corporate"
		} else if a.MTLAXBalance > 0 && a.MTLAPBalance == 0 {
			accountType = "synthetic"
		}

		grade := ""
		if a.ReputationScore > 0 {
			grade = reputation.ScoreToGrade(a.ReputationScore)
		}

		return AccountListItem{
			ID:              a.AccountID,
			Name:            a.Name,
			Type:            accountType,
			MTLAPBalance:    a.MTLAPBalance,
			MTLACBalance:    a.MTLACBalance,
			MTLAXBalance:    a.MTLAXBalance,
			TotalXLMValue:   a.TotalXLMValue,
			ReputationScore: a.ReputationScore,
			ReputationGrade: grade,
			IsCouncilReady:  a.IsCouncilReady,
			ReceivedVotes:   a.ReceivedVotes,
		}
	})

	return items, total, nil
}

// GetAccount handles GET /api/v1/accounts/{id}.
//
//	@Summary		Get account detail
//	@Description	Returns full account detail including metadata, trustlines, LP shares, trust ratings, and reputation
//	@Tags			accounts
//	@Produce		json
//	@Param			id	path		string	true	"Stellar account ID"
//	@Success		200	{object}	AccountDetailResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/accounts/{id} [get]
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account ID is required")
		return
	}

	if !isValidStellarID(accountID) {
		writeError(w, http.StatusBadRequest, "invalid Stellar account ID format")
		return
	}

	// Check if account exists
	exists, err := h.accounts.AccountExists(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to check account existence", "account_id", accountID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch account")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	// Fetch account metadata
	meta, err := h.accounts.GetAccountMetadata(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch account metadata", "account_id", accountID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch account")
		return
	}

	// Fetch account info (balances, XLM value)
	accountInfo, err := h.accounts.GetAccountInfo(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch account info", "account_id", accountID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch account")
		return
	}

	name := meta.Name
	if name == "" && len(accountID) >= 12 {
		name = accountID[:6] + "..." + accountID[len(accountID)-6:]
	}

	resp := AccountDetailResponse{
		ID:            accountID,
		Name:          name,
		About:         meta.About,
		Websites:      meta.Websites,
		Tags:          meta.Tags,
		IsCorporate:   accountInfo.MTLACBalance > 0,
		TotalXLMValue: accountInfo.TotalXLMValue,
	}

	// Fetch trustlines (account balances)
	balances, err := h.accounts.GetAccountBalances(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch account balances", "account_id", accountID, "error", err)
	} else if len(balances) > 0 {
		resp.Trustlines = lo.Map(balances, func(b repository.BalanceRow, _ int) TrustlineResponse {
			return TrustlineResponse{
				AssetCode:   b.AssetCode,
				AssetIssuer: b.AssetIssuer,
				Balance:     fmt.Sprintf("%.7f", b.Balance),
			}
		})
	}

	// Fetch trust ratings
	trustRating, err := h.accounts.GetTrustRatings(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch trust ratings", "account_id", accountID, "error", err)
	} else if trustRating != nil && trustRating.Total > 0 {
		resp.TrustRating = &TrustRatingResponse{
			CountA: trustRating.CountA,
			CountB: trustRating.CountB,
			CountC: trustRating.CountC,
			CountD: trustRating.CountD,
			Total:  trustRating.Total,
			Score:  trustRating.Score,
			Grade:  bsn.CalculateTrustGrade(trustRating.Score),
		}
	}

	// Fetch LP shares
	lpRows, err := h.accounts.GetLPShares(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch LP shares", "account_id", accountID, "error", err)
	} else if len(lpRows) > 0 {
		resp.LPShares = lo.Map(lpRows, func(lp repository.LPShareRow, _ int) LPShareResponse {
			sharePercent := "0%"
			if lp.TotalShares > 0 {
				pct := (lp.ShareBalance / lp.TotalShares) * 100
				if pct < 0.01 {
					sharePercent = "<0.01%"
				} else {
					sharePercent = fmt.Sprintf("%.2f%%", pct)
				}
			}

			shareRatio := float64(0)
			if lp.TotalShares > 0 {
				shareRatio = lp.ShareBalance / lp.TotalShares
			}

			return LPShareResponse{
				PoolID:       lp.PoolID,
				ShareBalance: fmt.Sprintf("%.7f", lp.ShareBalance),
				SharePercent: sharePercent,
				ReserveA: ReserveResponse{
					AssetCode:   lp.ReserveACode,
					AssetIssuer: lp.ReserveAIssuer,
					Amount:      fmt.Sprintf("%.4f", lp.ReserveAAmount*shareRatio),
				},
				ReserveB: ReserveResponse{
					AssetCode:   lp.ReserveBCode,
					AssetIssuer: lp.ReserveBIssuer,
					Amount:      fmt.Sprintf("%.4f", lp.ReserveBAmount*shareRatio),
				},
				XLMValue: lp.XLMValue,
			}
		})
	}

	// Fetch reputation
	if h.reputation != nil {
		score, err := h.reputation.GetScore(ctx, accountID)
		if err != nil {
			slog.Warn("api: failed to fetch reputation score", "account_id", accountID, "error", err)
		} else if score != nil {
			resp.Reputation = convertReputationScore(score)
		}
	}

	// Fetch relationships
	relationships, err := h.accounts.GetRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch relationships", "account_id", accountID, "error", err)
	}

	confirmed, err := h.accounts.GetConfirmedRelationships(ctx, accountID)
	if err != nil {
		slog.Error("api: failed to fetch confirmed relationships", "account_id", accountID, "error", err)
		confirmed = make(map[string]bool)
	}

	if len(relationships) > 0 {
		categories := bsn.GroupRelationships(accountID, relationships, confirmed)
		resp.Categories = lo.Map(categories, func(cat model.RelationshipCategory, _ int) RelationshipCategoryResponse {
			if cat.IsEmpty {
				return RelationshipCategoryResponse{
					Name:          cat.Name,
					Color:         cat.Color,
					Relationships: []RelationshipResponse{},
				}
			}
			return RelationshipCategoryResponse{
				Name:  cat.Name,
				Color: cat.Color,
				Relationships: lo.Map(cat.Relationships, func(rel model.Relationship, _ int) RelationshipResponse {
					return RelationshipResponse{
						Type:        rel.Type,
						TargetID:    rel.TargetID,
						TargetName:  rel.TargetName,
						Direction:   rel.Direction,
						IsMutual:    rel.IsMutual,
						IsConfirmed: rel.IsConfirmed,
					}
				}),
			}
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func convertReputationScore(score *model.ReputationScore) *ReputationResponse {
	if score == nil {
		return nil
	}
	return &ReputationResponse{
		WeightedScore: score.WeightedScore,
		BaseScore:     score.BaseScore,
		Grade:         score.Grade,
		RatingCountA:  score.RatingCountA,
		RatingCountB:  score.RatingCountB,
		RatingCountC:  score.RatingCountC,
		RatingCountD:  score.RatingCountD,
		TotalRatings:  score.TotalRatings,
		TotalWeight:   score.TotalWeight,
	}
}
