package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/service"
	"github.com/samber/lo"
)

// AccountData holds data for the account detail page template.
type AccountData struct {
	Account      *model.AccountDetail
	Operations   *model.OperationsPage
	AccountNames map[string]string // Map of account ID to name for linked accounts
}

// relationshipCategoryDef defines a relationship category.
type relationshipCategoryDef struct {
	Name  string
	Color string
	Types []string
}

// categoryDefinitions holds all category definitions in display order.
var categoryDefinitions = []relationshipCategoryDef{
	{
		Name:  "FAMILY",
		Color: "#f85149",
		Types: []string{"OneFamily", "Spouse", "Guardian", "Ward", "Sympathy", "Love", "Divorce"},
	},
	{
		Name:  "WORK",
		Color: "#58a6ff",
		Types: []string{"Employer", "Employee", "Contractor", "Client"},
	},
	{
		Name:  "NETWORK",
		Color: "#a371f7",
		Types: []string{"Partnership", "Collaboration", "MyPart", "PartOf", "RecommendToMTLA"},
	},
	{
		Name:  "OWNERSHIP",
		Color: "#f0b429",
		Types: []string{"OwnershipFull", "OwnershipMajority", "OwnershipMinority", "Owner", "OwnerMajority", "OwnerMinority"},
	},
	{
		Name:  "SOCIAL",
		Color: "#00ff88",
		Types: []string{"WelcomeGuest", "FactionMember"},
	},
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

	// Process relationships into categories
	account.Categories = groupRelationships(accountID, relationships, confirmed)

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
			Grade:   calculateTrustGrade(trustRating.Score),
			Percent: int((trustRating.Score / 4.0) * 100),
		}
	}

	data := AccountData{
		Account:      account,
		Operations:   operations,
		AccountNames: accountNames,
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

// complementaryPairs maps relationship types to their complementary type.
// When both sides of a pair exist, the relationship is "confirmed".
var complementaryPairs = map[string]string{
	"MyPart":            "PartOf",
	"PartOf":            "MyPart",
	"Guardian":          "Ward",
	"Ward":              "Guardian",
	"OwnershipFull":     "Owner",
	"Owner":             "OwnershipFull",
	"OwnershipMajority": "OwnerMajority",
	"OwnerMajority":     "OwnershipMajority",
	"OwnershipMinority": "OwnerMinority",
	"OwnerMinority":     "OwnershipMinority",
	"Employer":          "Employee",
	"Employee":          "Employer",
}

// symmetricTypes are relationship types where both parties must declare for display.
// One-way declarations are hidden.
var symmetricTypes = map[string]bool{
	"FactionMember": true,
	"Partnership":   true,
	"Collaboration": true,
	"Spouse":        true,
	"OneFamily":     true,
}

// groupRelationships organizes relationships into display categories.
// It merges complementary pairs and deduplicates mutual symmetric relationships.
// For symmetric types (FactionMember, Partnership, etc.), one-way relationships are hidden.
func groupRelationships(accountID string, rows []repository.RelationshipRow, confirmed map[string]bool) []model.RelationshipCategory {
	// Build maps for relationship lookup
	// Key: "otherAccountID:type", value: row
	outgoingMap := make(map[string]repository.RelationshipRow)
	incomingMap := make(map[string]repository.RelationshipRow)

	for _, row := range rows {
		var otherID string
		if row.Direction == "outgoing" {
			otherID = row.TargetAccountID
			key := fmt.Sprintf("%s:%s", otherID, row.RelationType)
			outgoingMap[key] = row
		} else {
			otherID = row.SourceAccountID
			key := fmt.Sprintf("%s:%s", otherID, row.RelationType)
			incomingMap[key] = row
		}
	}

	// Build type-to-category lookup
	typeToCat := make(map[string]int)
	for i, cat := range categoryDefinitions {
		for _, t := range cat.Types {
			typeToCat[t] = i
		}
	}

	// Track which relationships we've already processed (to avoid duplicates)
	processed := make(map[string]bool)

	// Group relationships by category
	categoryRels := make([][]model.Relationship, len(categoryDefinitions))
	for i := range categoryRels {
		categoryRels[i] = []model.Relationship{}
	}

	for _, row := range rows {
		// Determine the "other" account
		var otherID string
		if row.Direction == "outgoing" {
			otherID = row.TargetAccountID
		} else {
			otherID = row.SourceAccountID
		}

		// Check if this is a complementary pair
		if complement, ok := complementaryPairs[row.RelationType]; ok {
			// Create a canonical key for this pair using sorted type names
			typeA, typeB := row.RelationType, complement
			if typeB < typeA {
				typeA, typeB = typeB, typeA
			}
			pairKey := fmt.Sprintf("%s:%s:%s:complementary", otherID, typeA, typeB)
			if processed[pairKey] {
				continue // Already processed this pair
			}

			// Check if complementary relationship exists (current type + complement)
			currentKey := fmt.Sprintf("%s:%s", otherID, row.RelationType)
			complementKey := fmt.Sprintf("%s:%s", otherID, complement)

			hasCurrent := false
			hasComplement := false

			if _, exists := outgoingMap[currentKey]; exists {
				hasCurrent = true
			}
			if _, exists := incomingMap[currentKey]; exists {
				hasCurrent = true
			}
			if _, exists := outgoingMap[complementKey]; exists {
				hasComplement = true
			}
			if _, exists := incomingMap[complementKey]; exists {
				hasComplement = true
			}

			catIdx, ok := typeToCat[row.RelationType]
			if !ok {
				continue
			}

			// Determine confirmation status
			isConfirmed := hasCurrent && hasComplement

			rel := model.Relationship{
				Type:        row.RelationType,
				TargetID:    otherID,
				TargetName:  row.TargetName,
				Direction:   row.Direction,
				IsMutual:    false,
				IsConfirmed: isConfirmed,
			}

			categoryRels[catIdx] = append(categoryRels[catIdx], rel)
			processed[pairKey] = true
			continue
		}

		catIdx, ok := typeToCat[row.RelationType]
		if !ok {
			continue
		}

		// Check if mutual (same type exists in both directions)
		mutualKey := fmt.Sprintf("%s:%s", otherID, row.RelationType)
		_, hasOutgoing := outgoingMap[mutualKey]
		_, hasIncoming := incomingMap[mutualKey]
		isMutual := hasOutgoing && hasIncoming

		// For symmetric types, skip if not mutual (hide one-way declarations)
		if symmetricTypes[row.RelationType] && !isMutual {
			continue
		}

		// If mutual, only process once (from outgoing direction)
		if isMutual {
			dedupeKey := fmt.Sprintf("%s:%s:mutual", otherID, row.RelationType)
			if processed[dedupeKey] {
				continue
			}
			processed[dedupeKey] = true
		}

		// Check if confirmed (from confirmed_relationships view)
		isConfirmed := false
		if row.Direction == "outgoing" {
			key := fmt.Sprintf("%s:%s:%s", accountID, row.TargetAccountID, row.RelationType)
			isConfirmed = confirmed[key]
		} else {
			key := fmt.Sprintf("%s:%s:%s", row.SourceAccountID, accountID, row.RelationType)
			isConfirmed = confirmed[key]
		}

		rel := model.Relationship{
			Type:        row.RelationType,
			TargetID:    otherID,
			TargetName:  row.TargetName,
			Direction:   row.Direction,
			IsMutual:    isMutual,
			IsConfirmed: isConfirmed,
		}

		categoryRels[catIdx] = append(categoryRels[catIdx], rel)
	}

	// Sort relationships: confirmed/mutual first, then unconfirmed.
	// Prioritizes verified relationships (where both parties declared) for user trust.
	for i := range categoryRels {
		sort.SliceStable(categoryRels[i], func(a, b int) bool {
			aConfirmed := categoryRels[i][a].IsConfirmed || categoryRels[i][a].IsMutual
			bConfirmed := categoryRels[i][b].IsConfirmed || categoryRels[i][b].IsMutual
			if aConfirmed != bConfirmed {
				return aConfirmed
			}
			return false // stable sort: preserve insertion order within groups
		})
	}

	// Build final categories
	categories := lo.Map(categoryDefinitions, func(def relationshipCategoryDef, i int) model.RelationshipCategory {
		return model.RelationshipCategory{
			Name:          def.Name,
			Color:         def.Color,
			Relationships: categoryRels[i],
			IsEmpty:       len(categoryRels[i]) == 0,
		}
	})

	return categories
}

// calculateTrustGrade converts a numeric score (1-4) to a letter grade.
func calculateTrustGrade(score float64) string {
	switch {
	case score >= 3.5:
		return "A"
	case score >= 3.0:
		return "A-"
	case score >= 2.5:
		return "B+"
	case score >= 2.0:
		return "B"
	case score >= 1.5:
		return "C+"
	case score >= 1.0:
		return "C"
	default:
		return "D"
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
