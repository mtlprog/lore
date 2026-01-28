package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/mtlprog/lore/internal/config"
	"github.com/stellar/go/clients/horizonclient"
)

// AssociationTag represents a tag from the Association account.
type AssociationTag struct {
	TagName         string
	TagIndex        string
	TargetAccountID string
}

// syncAssociationTags fetches tags from the Association account.
func (s *Syncer) syncAssociationTags(ctx context.Context) error {
	// Fetch the Association account (TokenIssuer)
	acc, err := s.horizon.AccountDetail(horizonclient.AccountRequest{AccountID: config.TokenIssuer})
	if err != nil {
		return fmt.Errorf("fetch association account: %w", err)
	}

	// Parse tags from ManageData
	tags := parseAssociationTags(acc.Data)

	// Group by tag name and upsert
	tagsByName := make(map[string][]AssociationTag)
	for _, tag := range tags {
		tagsByName[tag.TagName] = append(tagsByName[tag.TagName], tag)
	}

	for tagName, tagList := range tagsByName {
		if err := s.repo.UpsertAssociationTags(ctx, tagName, tagList); err != nil {
			return fmt.Errorf("upsert association tags: %w", err)
		}
	}

	return nil
}

// Known association tag types
var associationTagTypes = []string{"Program", "Faction"}

// parseAssociationTags extracts tags from Association account ManageData.
func parseAssociationTags(rawData map[string]string) []AssociationTag {
	var tags []AssociationTag

	for key := range rawData {
		for _, tagType := range associationTagTypes {
			if !strings.HasPrefix(key, tagType) {
				continue
			}

			// Extract account ID and index
			rest := key[len(tagType):]
			if len(rest) < 56 {
				continue
			}

			targetID := rest[:56]
			if !strings.HasPrefix(targetID, "G") {
				continue
			}

			index := "0"
			if len(rest) > 56 {
				index = rest[56:]
			}

			tags = append(tags, AssociationTag{
				TagName:         tagType,
				TagIndex:        index,
				TargetAccountID: targetID,
			})
		}
	}

	return tags
}
