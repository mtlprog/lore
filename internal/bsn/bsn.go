package bsn

import (
	"fmt"
	"sort"

	"github.com/mtlprog/lore/internal/model"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/samber/lo"
)

// CategoryDef defines a relationship category.
type CategoryDef struct {
	Name  string
	Color string
	Types []string
}

var categoryDefinitions = []CategoryDef{
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

var symmetricTypes = map[string]bool{
	"FactionMember": true,
	"Partnership":   true,
	"Collaboration": true,
	"Spouse":        true,
	"OneFamily":     true,
}

// GroupRelationships organizes relationships into display categories.
// It merges complementary pairs and deduplicates mutual symmetric relationships.
// For symmetric types (FactionMember, Partnership, etc.), one-way relationships are hidden.
func GroupRelationships(accountID string, rows []repository.RelationshipRow, confirmed map[string]bool) []model.RelationshipCategory {
	// Build maps for relationship lookup
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
		var otherID string
		if row.Direction == "outgoing" {
			otherID = row.TargetAccountID
		} else {
			otherID = row.SourceAccountID
		}

		// Check if this is a complementary pair
		if complement, ok := complementaryPairs[row.RelationType]; ok {
			typeA, typeB := row.RelationType, complement
			if typeB < typeA {
				typeA, typeB = typeB, typeA
			}
			pairKey := fmt.Sprintf("%s:%s:%s:complementary", otherID, typeA, typeB)
			if processed[pairKey] {
				continue
			}

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

		mutualKey := fmt.Sprintf("%s:%s", otherID, row.RelationType)
		_, hasOutgoing := outgoingMap[mutualKey]
		_, hasIncoming := incomingMap[mutualKey]
		isMutual := hasOutgoing && hasIncoming

		if symmetricTypes[row.RelationType] && !isMutual {
			continue
		}

		if isMutual {
			dedupeKey := fmt.Sprintf("%s:%s:mutual", otherID, row.RelationType)
			if processed[dedupeKey] {
				continue
			}
			processed[dedupeKey] = true
		}

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
	for i := range categoryRels {
		sort.SliceStable(categoryRels[i], func(a, b int) bool {
			aConfirmed := categoryRels[i][a].IsConfirmed || categoryRels[i][a].IsMutual
			bConfirmed := categoryRels[i][b].IsConfirmed || categoryRels[i][b].IsMutual
			if aConfirmed != bConfirmed {
				return aConfirmed
			}
			return false
		})
	}

	// Build final categories
	categories := lo.Map(categoryDefinitions, func(def CategoryDef, i int) model.RelationshipCategory {
		return model.RelationshipCategory{
			Name:          def.Name,
			Color:         def.Color,
			Relationships: categoryRels[i],
			IsEmpty:       len(categoryRels[i]) == 0,
		}
	})

	return categories
}

// CalculateTrustGrade converts a numeric score (1-4) to a letter grade.
func CalculateTrustGrade(score float64) string {
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
