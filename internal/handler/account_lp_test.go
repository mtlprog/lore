package handler

import (
	"testing"

	"github.com/mtlprog/lore/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestConvertLPShares(t *testing.T) {
	tests := []struct {
		name     string
		input    []repository.LPShareRow
		expected int // number of results
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty slice returns nil",
			input:    []repository.LPShareRow{},
			expected: 0,
		},
		{
			name: "single LP share",
			input: []repository.LPShareRow{
				{
					PoolID:         "pool123",
					ShareBalance:   100.0,
					TotalShares:    1000.0,
					ReserveACode:   "MTL",
					ReserveAIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
					ReserveAAmount: 10000.0,
					ReserveBCode:   "XLM",
					ReserveBIssuer: "",
					ReserveBAmount: 50000.0,
					XLMValue:       5000.0,
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLPShares(tt.input)
			if tt.expected == 0 {
				assert.Nil(t, result)
			} else {
				assert.Len(t, result, tt.expected)
			}
		})
	}
}

func TestConvertLPShares_SharePercentage(t *testing.T) {
	tests := []struct {
		name            string
		shareBalance    float64
		totalShares     float64
		expectedPercent string
	}{
		{
			name:            "10% share",
			shareBalance:    100.0,
			totalShares:     1000.0,
			expectedPercent: "10.00%",
		},
		{
			name:            "small percentage less than 0.01%",
			shareBalance:    1.0,
			totalShares:     100000.0,
			expectedPercent: "<0.01%",
		},
		{
			name:            "exactly 0.01%",
			shareBalance:    1.0,
			totalShares:     10000.0,
			expectedPercent: "0.01%",
		},
		{
			name:            "zero total shares (division by zero protection)",
			shareBalance:    100.0,
			totalShares:     0.0,
			expectedPercent: "0%",
		},
		{
			name:            "fractional percentage",
			shareBalance:    123.456,
			totalShares:     10000.0,
			expectedPercent: "1.23%",
		},
		{
			name:            "100% ownership",
			shareBalance:    1000.0,
			totalShares:     1000.0,
			expectedPercent: "100.00%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []repository.LPShareRow{
				{
					PoolID:       "testpool",
					ShareBalance: tt.shareBalance,
					TotalShares:  tt.totalShares,
				},
			}

			result := convertLPShares(input)
			assert.Len(t, result, 1)
			assert.Equal(t, tt.expectedPercent, result[0].SharePercent)
		})
	}
}

func TestConvertLPShares_ProportionalReserves(t *testing.T) {
	tests := []struct {
		name           string
		shareBalance   float64
		totalShares    float64
		reserveAAmount float64
		reserveBAmount float64
		expectedA      string
		expectedB      string
	}{
		{
			name:           "10% of pool reserves",
			shareBalance:   100.0,
			totalShares:    1000.0,
			reserveAAmount: 10000.0,
			reserveBAmount: 50000.0,
			expectedA:      "1000.0000",
			expectedB:      "5000.0000",
		},
		{
			name:           "zero total shares returns zero reserves",
			shareBalance:   100.0,
			totalShares:    0.0,
			reserveAAmount: 10000.0,
			reserveBAmount: 50000.0,
			expectedA:      "0.0000",
			expectedB:      "0.0000",
		},
		{
			name:           "full ownership",
			shareBalance:   1000.0,
			totalShares:    1000.0,
			reserveAAmount: 12345.6789,
			reserveBAmount: 98765.4321,
			expectedA:      "12345.6789",
			expectedB:      "98765.4321",
		},
		{
			name:           "small fractional ownership",
			shareBalance:   1.0,
			totalShares:    1000000.0,
			reserveAAmount: 1000000.0,
			reserveBAmount: 2000000.0,
			expectedA:      "1.0000",
			expectedB:      "2.0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []repository.LPShareRow{
				{
					PoolID:         "testpool",
					ShareBalance:   tt.shareBalance,
					TotalShares:    tt.totalShares,
					ReserveACode:   "MTL",
					ReserveAIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
					ReserveAAmount: tt.reserveAAmount,
					ReserveBCode:   "XLM",
					ReserveBIssuer: "",
					ReserveBAmount: tt.reserveBAmount,
				},
			}

			result := convertLPShares(input)
			assert.Len(t, result, 1)
			assert.Equal(t, tt.expectedA, result[0].ReserveA.Amount)
			assert.Equal(t, tt.expectedB, result[0].ReserveB.Amount)
		})
	}
}

func TestConvertLPShares_FieldMapping(t *testing.T) {
	input := []repository.LPShareRow{
		{
			PoolID:         "abc123poolid",
			ShareBalance:   500.1234567,
			TotalShares:    1000.0,
			ReserveACode:   "EURMTL",
			ReserveAIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
			ReserveAAmount: 20000.0,
			ReserveBCode:   "XLM",
			ReserveBIssuer: "",
			ReserveBAmount: 100000.0,
			XLMValue:       50000.12345,
		},
	}

	result := convertLPShares(input)

	assert.Len(t, result, 1)
	lp := result[0]

	// Verify all fields are mapped correctly
	assert.Equal(t, "abc123poolid", lp.PoolID)
	assert.Equal(t, "500.1234567", lp.ShareBalance)
	assert.Equal(t, "50.01%", lp.SharePercent)
	assert.Equal(t, 50000.12345, lp.XLMValue)

	// Reserve A
	assert.Equal(t, "EURMTL", lp.ReserveA.AssetCode)
	assert.Equal(t, "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V", lp.ReserveA.AssetIssuer)

	// Reserve B
	assert.Equal(t, "XLM", lp.ReserveB.AssetCode)
	assert.Equal(t, "", lp.ReserveB.AssetIssuer)
}

func TestConvertLPShares_MultipleShares(t *testing.T) {
	input := []repository.LPShareRow{
		{
			PoolID:       "pool1",
			ShareBalance: 100.0,
			TotalShares:  1000.0,
			ReserveACode: "MTL",
			ReserveBCode: "XLM",
		},
		{
			PoolID:       "pool2",
			ShareBalance: 50.0,
			TotalShares:  500.0,
			ReserveACode: "EURMTL",
			ReserveBCode: "USDC",
		},
		{
			PoolID:       "pool3",
			ShareBalance: 1.0,
			TotalShares:  10000000.0, // Very small share
			ReserveACode: "EURDEBT",
			ReserveBCode: "EUR",
		},
	}

	result := convertLPShares(input)

	assert.Len(t, result, 3)

	// Verify order is preserved
	assert.Equal(t, "pool1", result[0].PoolID)
	assert.Equal(t, "pool2", result[1].PoolID)
	assert.Equal(t, "pool3", result[2].PoolID)

	// Verify percentages
	assert.Equal(t, "10.00%", result[0].SharePercent)
	assert.Equal(t, "10.00%", result[1].SharePercent)
	assert.Equal(t, "<0.01%", result[2].SharePercent) // Very small percentage
}
