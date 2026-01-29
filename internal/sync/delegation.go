package sync

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

// calculateDelegations computes delegation chains and council votes.
func (s *Syncer) calculateDelegations(ctx context.Context) error {
	if err := s.repo.ResetDelegations(ctx); err != nil {
		return fmt.Errorf("reset delegations: %w", err)
	}

	accounts, err := s.repo.GetAllDelegationInfo(ctx)
	if err != nil {
		return fmt.Errorf("get delegation info: %w", err)
	}

	// Build lookup map using lo.KeyBy
	accountMap := lo.KeyBy(accounts, func(acc DelegationInfo) string {
		return acc.AccountID
	})
	// Convert to pointer map for mutations
	accountPtrMap := make(map[string]*DelegationInfo, len(accounts))
	for i := range accounts {
		accountPtrMap[accounts[i].AccountID] = &accounts[i]
	}

	var dbErrors []string

	// Step 3: For each account that delegates, trace the chain
	for _, acc := range accounts {
		if acc.DelegateTo == nil {
			continue
		}

		delegateAcc, exists := accountMap[*acc.DelegateTo]
		if !exists || delegateAcc.MTLAPBalance.IsZero() {
			if err := s.repo.SetDelegationError(ctx, acc.AccountID, true); err != nil {
				s.logger.Error("failed to set delegation error", "account_id", acc.AccountID, "error", err)
				dbErrors = append(dbErrors, acc.AccountID)
			}
			continue
		}

		path, hasCycle := traceDelegationChain(acc.AccountID, accountPtrMap)
		if hasCycle {
			for _, cycleAccID := range path {
				if err := s.repo.SetCycleError(ctx, cycleAccID, path); err != nil {
					s.logger.Error("failed to set cycle error", "account_id", cycleAccID, "error", err)
					dbErrors = append(dbErrors, cycleAccID)
				}
			}
		}
	}

	// Step 4: Calculate received votes for council-ready accounts
	councilReadyAccounts := lo.Filter(accounts, func(acc DelegationInfo, _ int) bool {
		return acc.CouncilReady
	})

	for _, acc := range councilReadyAccounts {
		votes := calculateReceivedVotes(acc.AccountID, accountPtrMap)
		if err := s.repo.SetReceivedVotes(ctx, acc.AccountID, votes); err != nil {
			s.logger.Error("failed to set received votes", "account_id", acc.AccountID, "error", err)
			dbErrors = append(dbErrors, acc.AccountID)
		}
	}

	if len(dbErrors) > 0 {
		s.logger.Error("delegation calculation had database errors",
			"failed_count", len(dbErrors),
			"failed_accounts", dbErrors[:min(10, len(dbErrors))],
		)
		if len(dbErrors) > 10 {
			return fmt.Errorf("too many database errors during delegation calculation: %d failures", len(dbErrors))
		}
	}

	return nil
}

// traceDelegationChain follows the delegation chain and detects cycles.
func traceDelegationChain(startID string, accountMap map[string]*DelegationInfo) ([]string, bool) {
	visited := make(map[string]bool)
	var path []string

	current := startID
	for {
		if visited[current] {
			cycleStart := lo.IndexOf(path, current)
			if cycleStart >= 0 {
				return path[cycleStart:], true
			}
			return path, true
		}

		visited[current] = true
		path = append(path, current)

		acc, exists := accountMap[current]
		if !exists || acc.DelegateTo == nil {
			break
		}

		current = *acc.DelegateTo
	}

	return path, false
}

// calculateReceivedVotes sums MTLAP from all accounts that delegate council votes to the target.
// Council delegation follows the mtla_c_delegate chain, not mtla_delegate.
func calculateReceivedVotes(targetID string, accountMap map[string]*DelegationInfo) int {
	totalVotes := decimal.Zero

	for _, acc := range accountMap {
		if acc.AccountID == targetID {
			continue
		}

		finalTarget := getFinalCouncilDelegationTarget(acc.AccountID, accountMap)
		if finalTarget == targetID {
			totalVotes = totalVotes.Add(acc.MTLAPBalance)
		}
	}

	return int(totalVotes.IntPart())
}

// getFinalDelegationTarget follows the delegation chain (mtla_delegate) to find the final target.
func getFinalDelegationTarget(startID string, accountMap map[string]*DelegationInfo) string {
	visited := make(map[string]bool)
	current := startID

	for {
		if visited[current] {
			return ""
		}
		visited[current] = true

		acc, exists := accountMap[current]
		if !exists {
			return ""
		}

		if acc.DelegateTo == nil {
			if acc.CouncilReady {
				return current
			}
			return ""
		}

		delegateAcc, exists := accountMap[*acc.DelegateTo]
		if !exists || delegateAcc.MTLAPBalance.IsZero() {
			return ""
		}

		current = *acc.DelegateTo
	}
}

// getFinalCouncilDelegationTarget follows the council delegation chain (mtla_c_delegate) to find the final target.
// The chain ends when we reach a council-ready account (mtla_c_delegate == "ready").
func getFinalCouncilDelegationTarget(startID string, accountMap map[string]*DelegationInfo) string {
	visited := make(map[string]bool)
	current := startID

	for {
		if visited[current] {
			return "" // cycle detected
		}
		visited[current] = true

		acc, exists := accountMap[current]
		if !exists {
			return ""
		}

		// If this account is council-ready, it's the final target
		if acc.CouncilReady {
			return current
		}

		// If no council delegation, the chain ends without a valid target
		if acc.CouncilDelegateTo == nil {
			return ""
		}

		// Follow the council delegation chain
		delegateAcc, exists := accountMap[*acc.CouncilDelegateTo]
		if !exists {
			return "" // target doesn't exist
		}

		// Note: We don't check MTLAPBalance here - accounts can delegate council votes
		// even if they have 0 MTLAP, as long as the chain eventually reaches a council-ready target
		_ = delegateAcc // just to avoid unused variable warning

		current = *acc.CouncilDelegateTo
	}
}
