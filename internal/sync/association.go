package sync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mtlprog/lore/internal/config"
	"github.com/samber/lo"
	"github.com/stellar/go/clients/horizonclient"
)

// tagNameStrings maps string to TagName for parsing.
var tagNameStrings = map[string]TagName{
	"Program": TagProgram,
	"Faction": TagFaction,
}

// tagNamePrefixes for parsing tag keys.
var tagNamePrefixes = []string{"Program", "Faction"}

// syncAssociationTags fetches tags from the Association account.
func (s *Syncer) syncAssociationTags(ctx context.Context) error {
	acc, err := s.horizon.AccountDetail(horizonclient.AccountRequest{AccountID: config.TokenIssuer})
	if err != nil {
		return fmt.Errorf("fetch association account: %w", err)
	}

	tags := parseAssociationTags(acc.Data)

	// Group by tag name using lo.GroupBy
	tagsByName := lo.GroupBy(tags, func(tag AssociationTag) TagName {
		return tag.TagName
	})

	for tagName, tagList := range tagsByName {
		if err := s.repo.UpsertAssociationTags(ctx, tagName, tagList); err != nil {
			return fmt.Errorf("upsert association tags: %w", err)
		}
	}

	return nil
}

// parseAssociationTags extracts tags from Association account ManageData.
func parseAssociationTags(rawData map[string]string) []AssociationTag {
	var tags []AssociationTag

	for key := range rawData {
		for _, prefix := range tagNamePrefixes {
			if !strings.HasPrefix(key, prefix) {
				continue
			}

			rest := key[len(prefix):]
			if len(rest) < 56 {
				continue
			}

			targetID := rest[:56]
			if !strings.HasPrefix(targetID, "G") {
				continue
			}

			index := 0
			if len(rest) > 56 {
				var err error
				index, err = strconv.Atoi(rest[56:])
				if err != nil {
					continue
				}
			}

			tagName, ok := tagNameStrings[prefix]
			if !ok {
				continue
			}

			tags = append(tags, AssociationTag{
				TagName:         tagName,
				TagIndex:        index,
				TargetAccountID: targetID,
			})
		}
	}

	return tags
}
