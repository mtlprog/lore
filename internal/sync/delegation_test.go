package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceDelegationChain(t *testing.T) {
	// Helper to create pointer to string
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name        string
		startID     string
		accounts    map[string]*DelegationInfo
		expectCycle bool
		expectLen   int
	}{
		{
			name:    "no delegation",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 1},
			},
			expectCycle: false,
			expectLen:   1,
		},
		{
			name:    "simple chain",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: 1},
				"C": {AccountID: "C", DelegateTo: nil, MTLAPBalance: 1, CouncilReady: true},
			},
			expectCycle: false,
			expectLen:   3,
		},
		{
			name:    "simple cycle",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: 1},
			},
			expectCycle: true,
			expectLen:   2,
		},
		{
			name:    "longer cycle",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: 1},
				"C": {AccountID: "C", DelegateTo: strPtr("A"), MTLAPBalance: 1},
			},
			expectCycle: true,
			expectLen:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, hasCycle := traceDelegationChain(tt.startID, tt.accounts)
			assert.Equal(t, tt.expectCycle, hasCycle)
			assert.Len(t, path, tt.expectLen)
		})
	}
}

func TestGetFinalDelegationTarget(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		startID  string
		accounts map[string]*DelegationInfo
		expected string
	}{
		{
			name:    "no delegation - council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 1, CouncilReady: true},
			},
			expected: "A",
		},
		{
			name:    "no delegation - not council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 1, CouncilReady: false},
			},
			expected: "",
		},
		{
			name:    "delegates to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: nil, MTLAPBalance: 1, CouncilReady: true},
			},
			expected: "B",
		},
		{
			name:    "chain to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: 1},
				"C": {AccountID: "C", DelegateTo: nil, MTLAPBalance: 1, CouncilReady: true},
			},
			expected: "C",
		},
		{
			name:    "cycle returns empty",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: 1},
			},
			expected: "",
		},
		{
			name:    "delegate without MTLAP returns empty",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: 1},
				"B": {AccountID: "B", DelegateTo: nil, MTLAPBalance: 0, CouncilReady: true},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFinalDelegationTarget(tt.startID, tt.accounts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateReceivedVotes(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		targetID string
		accounts map[string]*DelegationInfo
		expected int
	}{
		{
			name:     "no delegators",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 10, CouncilReady: true},
			},
			expected: 0,
		},
		{
			name:     "one direct delegator",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 10, CouncilReady: true},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: 5},
			},
			expected: 5,
		},
		{
			name:     "multiple delegators",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 10, CouncilReady: true},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: 5},
				"C": {AccountID: "C", DelegateTo: strPtr("A"), MTLAPBalance: 3},
			},
			expected: 8,
		},
		{
			name:     "transitive delegation",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: 10, CouncilReady: true},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: 5},
				"C": {AccountID: "C", DelegateTo: strPtr("B"), MTLAPBalance: 3},
			},
			expected: 8, // B(5) + C(3) both ultimately delegate to A
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateReceivedVotes(tt.targetID, tt.accounts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
