package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAssetType(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "1 character code",
			code:     "X",
			expected: "credit_alphanum4",
		},
		{
			name:     "2 character code",
			code:     "XL",
			expected: "credit_alphanum4",
		},
		{
			name:     "3 character code",
			code:     "XLM",
			expected: "credit_alphanum4",
		},
		{
			name:     "4 character code (boundary)",
			code:     "USDC",
			expected: "credit_alphanum4",
		},
		{
			name:     "5 character code (boundary)",
			code:     "MTLAP",
			expected: "credit_alphanum12",
		},
		{
			name:     "6 character code",
			code:     "EURMTL",
			expected: "credit_alphanum12",
		},
		{
			name:     "12 character code",
			code:     "ABCDEFGHIJKL",
			expected: "credit_alphanum12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAssetType(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
