package sync

import (
	"context"
	"fmt"
	"log/slog"
)

// DelegationInfo holds delegation data for an account.
type DelegationInfo struct {
	AccountID    string
	DelegateTo   *string
	MTLAPBalance float64
	CouncilReady bool
}

// calculateDelegations computes delegation chains and council votes.
func (s *Syncer) calculateDelegations(ctx context.Context) error {
	// Step 1: Reset all delegation-related fields
	if err := s.repo.ResetDelegations(ctx); err != nil {
		return fmt.Errorf("reset delegations: %w", err)
	}

	// Step 2: Get all accounts with their delegation info
	accounts, err := s.repo.GetAllDelegationInfo(ctx)
	if err != nil {
		return fmt.Errorf("get delegation info: %w", err)
	}

	// Build lookup maps
	accountMap := make(map[string]*DelegationInfo)
	for i := range accounts {
		accountMap[accounts[i].AccountID] = &accounts[i]
	}

	var dbErrors []string

	// Step 3: For each account that delegates, trace the chain
	for _, acc := range accounts {
		if acc.DelegateTo == nil {
			continue
		}

		// Check for delegation errors
		delegateAcc, exists := accountMap[*acc.DelegateTo]
		if !exists || delegateAcc.MTLAPBalance == 0 {
			// Delegating to non-MTLAP holder is an error
			if err := s.repo.SetDelegationError(ctx, acc.AccountID, true); err != nil {
				slog.Error("failed to set delegation error", "account_id", acc.AccountID, "error", err)
				dbErrors = append(dbErrors, acc.AccountID)
			}
			continue
		}

		// Trace the delegation chain to detect cycles
		path, hasCycle := traceDelegationChain(acc.AccountID, accountMap)
		if hasCycle {
			// Mark all accounts in the cycle
			for _, cycleAccID := range path {
				if err := s.repo.SetCycleError(ctx, cycleAccID, path); err != nil {
					slog.Error("failed to set cycle error", "account_id", cycleAccID, "error", err)
					dbErrors = append(dbErrors, cycleAccID)
				}
			}
		}
	}

	// Step 4: Calculate received votes for council-ready accounts
	// Only count votes from accounts without delegation errors or cycles
	for _, acc := range accounts {
		if !acc.CouncilReady {
			continue
		}

		// Sum MTLAP from all accounts that delegate to this one (directly or transitively)
		votes := calculateReceivedVotes(acc.AccountID, accountMap)

		if err := s.repo.SetReceivedVotes(ctx, acc.AccountID, votes); err != nil {
			slog.Error("failed to set received votes", "account_id", acc.AccountID, "error", err)
			dbErrors = append(dbErrors, acc.AccountID)
		}
	}

	if len(dbErrors) > 0 {
		slog.Error("delegation calculation had database errors", "failed_count", len(dbErrors))
		// Return error if too many failures
		if len(dbErrors) > 10 {
			return fmt.Errorf("too many database errors during delegation calculation: %d failures", len(dbErrors))
		}
	}

	return nil
}

// traceDelegationChain follows the delegation chain and detects cycles.
// Returns the path and whether a cycle was detected.
func traceDelegationChain(startID string, accountMap map[string]*DelegationInfo) ([]string, bool) {
	visited := make(map[string]bool)
	var path []string

	current := startID
	for {
		if visited[current] {
			// Found a cycle - extract the cycle portion
			cycleStart := -1
			for i, id := range path {
				if id == current {
					cycleStart = i
					break
				}
			}
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

// calculateReceivedVotes sums MTLAP from all accounts that delegate to the target.
func calculateReceivedVotes(targetID string, accountMap map[string]*DelegationInfo) int {
	// Find all accounts whose delegation chain ends at targetID
	totalVotes := 0.0

	for _, acc := range accountMap {
		if acc.AccountID == targetID {
			continue
		}

		// Follow delegation chain from this account
		finalTarget := getFinalDelegationTarget(acc.AccountID, accountMap)
		if finalTarget == targetID {
			totalVotes += acc.MTLAPBalance
		}
	}

	return int(totalVotes)
}

// getFinalDelegationTarget follows the delegation chain to find the final target.
// Returns empty string if chain leads to cycle or error.
func getFinalDelegationTarget(startID string, accountMap map[string]*DelegationInfo) string {
	visited := make(map[string]bool)
	current := startID

	for {
		if visited[current] {
			return "" // Cycle detected
		}
		visited[current] = true

		acc, exists := accountMap[current]
		if !exists {
			return "" // Account not found
		}

		if acc.DelegateTo == nil {
			// This account doesn't delegate, so it's the final target
			if acc.CouncilReady {
				return current
			}
			return "" // End of chain but not council ready
		}

		// Check if delegate exists and has MTLAP
		delegateAcc, exists := accountMap[*acc.DelegateTo]
		if !exists || delegateAcc.MTLAPBalance == 0 {
			return "" // Delegation error
		}

		current = *acc.DelegateTo
	}
}
