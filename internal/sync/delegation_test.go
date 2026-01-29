package sync

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTraceDelegationChain(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	one := decimal.NewFromInt(1)

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
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: one},
			},
			expectCycle: false,
			expectLen:   1,
		},
		{
			name:    "simple chain",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: one},
				"C": {AccountID: "C", DelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expectCycle: false,
			expectLen:   3,
		},
		{
			name:    "simple cycle",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: one},
			},
			expectCycle: true,
			expectLen:   2,
		},
		{
			name:    "longer cycle",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: one},
				"C": {AccountID: "C", DelegateTo: strPtr("A"), MTLAPBalance: one},
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
	one := decimal.NewFromInt(1)
	zero := decimal.Zero

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
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "A",
		},
		{
			name:    "no delegation - not council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: nil, MTLAPBalance: one, CouncilReady: false},
			},
			expected: "",
		},
		{
			name:    "delegates to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "B",
		},
		{
			name:    "chain to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: strPtr("C"), MTLAPBalance: one},
				"C": {AccountID: "C", DelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "C",
		},
		{
			name:    "cycle returns empty",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: strPtr("A"), MTLAPBalance: one},
			},
			expected: "",
		},
		{
			name:    "delegate without MTLAP returns empty",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", DelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", DelegateTo: nil, MTLAPBalance: zero, CouncilReady: true},
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
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: decimal.NewFromInt(10), CouncilReady: true},
			},
			expected: 0,
		},
		{
			name:     "one direct delegator",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: decimal.NewFromInt(10), CouncilReady: true},
				"B": {AccountID: "B", CouncilDelegateTo: strPtr("A"), MTLAPBalance: decimal.NewFromInt(5)},
			},
			expected: 5,
		},
		{
			name:     "multiple delegators",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: decimal.NewFromInt(10), CouncilReady: true},
				"B": {AccountID: "B", CouncilDelegateTo: strPtr("A"), MTLAPBalance: decimal.NewFromInt(5)},
				"C": {AccountID: "C", CouncilDelegateTo: strPtr("A"), MTLAPBalance: decimal.NewFromInt(3)},
			},
			expected: 8,
		},
		{
			name:     "transitive delegation",
			targetID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: decimal.NewFromInt(10), CouncilReady: true},
				"B": {AccountID: "B", CouncilDelegateTo: strPtr("A"), MTLAPBalance: decimal.NewFromInt(5), CouncilReady: false},
				"C": {AccountID: "C", CouncilDelegateTo: strPtr("B"), MTLAPBalance: decimal.NewFromInt(3), CouncilReady: false},
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

func TestGetFinalCouncilDelegationTarget(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	one := decimal.NewFromInt(1)

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
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "A",
		},
		{
			name:    "no delegation - not council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: nil, MTLAPBalance: one, CouncilReady: false},
			},
			expected: "",
		},
		{
			name:    "delegates to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", CouncilDelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "B",
		},
		{
			name:    "chain to council ready",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", CouncilDelegateTo: strPtr("C"), MTLAPBalance: one},
				"C": {AccountID: "C", CouncilDelegateTo: nil, MTLAPBalance: one, CouncilReady: true},
			},
			expected: "C",
		},
		{
			name:    "cycle returns empty",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", CouncilDelegateTo: strPtr("A"), MTLAPBalance: one},
			},
			expected: "",
		},
		{
			name:    "delegate without MTLAP still follows chain",
			startID: "A",
			accounts: map[string]*DelegationInfo{
				"A": {AccountID: "A", CouncilDelegateTo: strPtr("B"), MTLAPBalance: one},
				"B": {AccountID: "B", CouncilDelegateTo: nil, MTLAPBalance: decimal.Zero, CouncilReady: true},
			},
			expected: "B", // Unlike regular delegation, council delegation doesn't require MTLAP balance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFinalCouncilDelegationTarget(tt.startID, tt.accounts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
