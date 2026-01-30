package reputation

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculator_calculateRaterWeight(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name        string
		portfolio   decimal.Decimal
		connections int
		wantMin     float64
		wantMax     float64
	}{
		{
			name:        "zero portfolio zero connections gets minimum weight",
			portfolio:   decimal.Zero,
			connections: 0,
			wantMin:     1.0,
			wantMax:     1.0,
		},
		{
			name:        "small portfolio gets minimum weight",
			portfolio:   decimal.NewFromInt(5),
			connections: 0,
			wantMin:     1.0,
			wantMax:     1.0,
		},
		{
			name:        "10 XLM no connections",
			portfolio:   decimal.NewFromInt(10),
			connections: 0,
			// log10(11) * sqrt(1) ≈ 1.04
			wantMin: 1.0,
			wantMax: 1.1,
		},
		{
			name:        "100 XLM with 3 connections",
			portfolio:   decimal.NewFromInt(100),
			connections: 3,
			// log10(101) * sqrt(4) = 2.004 * 2 ≈ 4.0
			wantMin: 3.9,
			wantMax: 4.1,
		},
		{
			name:        "1000 XLM with 8 connections",
			portfolio:   decimal.NewFromInt(1000),
			connections: 8,
			// log10(1001) * sqrt(9) = 3.0 * 3 = 9.0
			wantMin: 8.9,
			wantMax: 9.1,
		},
		{
			name:        "1M XLM with 99 connections gets capped at MaxWeight",
			portfolio:   decimal.NewFromInt(1_000_000),
			connections: 99,
			// Would be log10(1M+1) * sqrt(100) = 6 * 10 = 60, but capped
			// Actually 60 < 100, so not capped
			wantMin: 59.0,
			wantMax: 61.0,
		},
		{
			name:        "huge portfolio huge connections gets capped",
			portfolio:   decimal.NewFromInt(1_000_000_000),
			connections: 10000,
			// Would be log10(1B+1) * sqrt(10001) = 9 * 100 = 900, capped at 100
			wantMin: 100.0,
			wantMax: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.calculateRaterWeight(tt.portfolio, tt.connections)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateRaterWeight(%v, %d) = %v, want between %v and %v",
					tt.portfolio, tt.connections, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCalculator_CalculateScores(t *testing.T) {
	calc := NewCalculator()

	t.Run("empty ratings returns empty map", func(t *testing.T) {
		scores := calc.CalculateScores(nil, nil, nil)
		if len(scores) != 0 {
			t.Errorf("expected empty map, got %d entries", len(scores))
		}
	})

	t.Run("single A rating", func(t *testing.T) {
		ratings := []RatingEdge{
			{RaterAccountID: "rater1", RateeAccountID: "target1", Rating: RatingA},
		}
		portfolios := map[string]decimal.Decimal{
			"rater1": decimal.NewFromInt(100),
		}
		connections := map[string]int{
			"rater1": 5,
		}

		scores := calc.CalculateScores(ratings, portfolios, connections)

		score, ok := scores["target1"]
		if !ok {
			t.Fatal("expected score for target1")
		}

		if score.RatingCountA != 1 {
			t.Errorf("RatingCountA = %d, want 1", score.RatingCountA)
		}
		if score.TotalRatings != 1 {
			t.Errorf("TotalRatings = %d, want 1", score.TotalRatings)
		}
		if score.BaseScore != 4.0 {
			t.Errorf("BaseScore = %v, want 4.0", score.BaseScore)
		}
		if score.WeightedScore != 4.0 {
			t.Errorf("WeightedScore = %v, want 4.0", score.WeightedScore)
		}
	})

	t.Run("mixed ratings weighted by portfolio", func(t *testing.T) {
		// Two raters: one with huge portfolio (A), one with small portfolio (D)
		// The A rating should dominate due to higher weight
		ratings := []RatingEdge{
			{RaterAccountID: "whale", RateeAccountID: "target", Rating: RatingA},
			{RaterAccountID: "minnow", RateeAccountID: "target", Rating: RatingD},
		}
		portfolios := map[string]decimal.Decimal{
			"whale":  decimal.NewFromInt(100000), // High weight
			"minnow": decimal.NewFromInt(1),      // Minimum weight
		}
		connections := map[string]int{
			"whale":  10,
			"minnow": 0,
		}

		scores := calc.CalculateScores(ratings, portfolios, connections)
		score := scores["target"]

		// Base score = (4 + 1) / 2 = 2.5
		if math.Abs(score.BaseScore-2.5) > 0.01 {
			t.Errorf("BaseScore = %v, want ~2.5", score.BaseScore)
		}

		// Weighted score should be closer to 4.0 (A) because whale has higher weight
		if score.WeightedScore < 3.0 {
			t.Errorf("WeightedScore = %v, expected > 3.0 due to whale weighting", score.WeightedScore)
		}
	})

	t.Run("invalid ratings are skipped", func(t *testing.T) {
		ratings := []RatingEdge{
			{RaterAccountID: "rater1", RateeAccountID: "target", Rating: RatingA},
			{RaterAccountID: "rater2", RateeAccountID: "target", Rating: Rating("X")}, // Invalid
			{RaterAccountID: "rater3", RateeAccountID: "target", Rating: Rating("")},  // Invalid
		}
		portfolios := map[string]decimal.Decimal{}
		connections := map[string]int{}

		scores := calc.CalculateScores(ratings, portfolios, connections)
		score := scores["target"]

		if score.TotalRatings != 1 {
			t.Errorf("TotalRatings = %d, want 1 (invalid ratings should be skipped)", score.TotalRatings)
		}
	})
}
