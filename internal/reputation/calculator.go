package reputation

import (
	"math"

	"github.com/shopspring/decimal"
)

// Calculator computes weighted reputation scores.
type Calculator struct {
	// Configuration parameters
	MaxWeight float64 // Maximum weight cap to prevent outliers (default: 100.0)
}

// NewCalculator creates a new reputation calculator with default settings.
func NewCalculator() *Calculator {
	return &Calculator{
		MaxWeight: 100.0,
	}
}

// CalculateScores computes reputation scores for all accounts with ratings.
// Returns a map of accountID -> Score.
func (c *Calculator) CalculateScores(
	ratings []RatingEdge,
	portfolios map[string]decimal.Decimal,
	connectionCounts map[string]int,
) map[string]*Score {
	// Group ratings by target account
	ratingsByTarget := make(map[string][]RatingEdge)
	for _, r := range ratings {
		ratingsByTarget[r.RateeAccountID] = append(ratingsByTarget[r.RateeAccountID], r)
	}

	scores := make(map[string]*Score)

	for accountID, accountRatings := range ratingsByTarget {
		score := c.calculateAccountScore(accountID, accountRatings, portfolios, connectionCounts)
		scores[accountID] = score
	}

	return scores
}

// calculateAccountScore computes the reputation score for a single account.
func (c *Calculator) calculateAccountScore(
	accountID string,
	ratings []RatingEdge,
	portfolios map[string]decimal.Decimal,
	connections map[string]int,
) *Score {
	score := &Score{
		AccountID: accountID,
	}

	if len(ratings) == 0 {
		return score
	}

	var totalWeightedScore float64
	var totalWeight float64
	var totalBaseScore float64

	for _, rating := range ratings {
		ratingValue := RatingToValue(rating.Rating)
		if ratingValue == 0 {
			continue // Skip invalid ratings
		}

		// Count rating types
		switch rating.Rating {
		case "A":
			score.RatingCountA++
		case "B":
			score.RatingCountB++
		case "C":
			score.RatingCountC++
		case "D":
			score.RatingCountD++
		}

		// Calculate rater weight based on portfolio and connections
		portfolio := portfolios[rating.RaterAccountID]
		connCount := connections[rating.RaterAccountID]
		weight := c.calculateRaterWeight(portfolio, connCount)

		totalWeightedScore += ratingValue * weight
		totalWeight += weight
		totalBaseScore += ratingValue
	}

	score.TotalRatings = score.RatingCountA + score.RatingCountB + score.RatingCountC + score.RatingCountD

	if score.TotalRatings > 0 {
		score.BaseScore = totalBaseScore / float64(score.TotalRatings)
	}

	if totalWeight > 0 {
		score.WeightedScore = totalWeightedScore / totalWeight
		score.TotalWeight = totalWeight
	}

	return score
}

// calculateRaterWeight computes the weight of a rater's vote.
// Weight = log10(portfolio_xlm + 1) * sqrt(connections + 1)
// This ensures:
// - Larger portfolios have more weight (but logarithmically scaled to prevent whale dominance)
// - More connected accounts have more weight (socially validated)
func (c *Calculator) calculateRaterWeight(portfolio decimal.Decimal, connections int) float64 {
	// Portfolio weight: log10 scaling
	// 1 XLM -> 0, 10 XLM -> 1, 100 XLM -> 2, 1000 XLM -> 3, etc.
	portfolioFloat, _ := portfolio.Float64()
	portfolioWeight := math.Log10(portfolioFloat + 1)

	// Connection weight: sqrt scaling
	// Rewards connected accounts but with diminishing returns
	connectionWeight := math.Sqrt(float64(connections + 1))

	// Combined weight
	weight := portfolioWeight * connectionWeight

	// Minimum weight of 1.0 (everyone's vote counts)
	if weight < 1.0 {
		weight = 1.0
	}

	// Cap at maximum to prevent outliers
	if weight > c.MaxWeight {
		weight = c.MaxWeight
	}

	return weight
}

// CalculateRaterWeight is exported for use in graph building.
func (c *Calculator) CalculateRaterWeight(portfolio decimal.Decimal, connections int) float64 {
	return c.calculateRaterWeight(portfolio, connections)
}
