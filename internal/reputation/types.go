package reputation

import (
	"time"

	"github.com/shopspring/decimal"
)

// Rating represents a reputation rating (A, B, C, or D).
type Rating string

// Valid rating values.
const (
	RatingA Rating = "A"
	RatingB Rating = "B"
	RatingC Rating = "C"
	RatingD Rating = "D"
)

// IsValid returns true if the rating is one of the valid values (A, B, C, D).
func (r Rating) IsValid() bool {
	switch r {
	case RatingA, RatingB, RatingC, RatingD:
		return true
	default:
		return false
	}
}

// Value converts the rating to its numeric value (A=4, B=3, C=2, D=1).
// Returns 0 for invalid ratings.
func (r Rating) Value() float64 {
	switch r {
	case RatingA:
		return 4.0
	case RatingB:
		return 3.0
	case RatingC:
		return 2.0
	case RatingD:
		return 1.0
	default:
		return 0.0
	}
}

// Score represents a calculated reputation score for an account.
type Score struct {
	AccountID     string
	WeightedScore float64 // 0.0-4.0 weighted by rater portfolio/connections
	BaseScore     float64 // 0.0-4.0 simple average
	RatingCountA  int
	RatingCountB  int
	RatingCountC  int
	RatingCountD  int
	TotalRatings  int
	TotalWeight   float64
	CalculatedAt  time.Time
}

// Grade converts a score (0-4) to a letter grade.
func (s *Score) Grade() string {
	return ScoreToGrade(s.WeightedScore)
}

// ScoreToGrade converts a numeric score (1-4) to a letter grade.
func ScoreToGrade(score float64) string {
	switch {
	case score >= 3.5:
		return "A"
	case score >= 3.0:
		return "A-"
	case score >= 2.5:
		return "B+"
	case score >= 2.0:
		return "B"
	case score >= 1.5:
		return "C+"
	case score >= 1.0:
		return "C"
	case score > 0:
		return "D"
	default:
		return "N/A"
	}
}

// RatingEdge represents a rating relationship from rater to ratee.
type RatingEdge struct {
	RaterAccountID string
	RateeAccountID string
	Rating         Rating
}

// RaterInfo contains information about a rater for weight calculation.
type RaterInfo struct {
	AccountID            string
	Name                 string
	Rating               Rating
	PortfolioXLM         decimal.Decimal // total_xlm_value
	ConfirmedConnections int             // count of confirmed relationships
	OwnScore             float64         // rater's own reputation score (for display)
}

// GraphNode represents a node in the reputation graph.
type GraphNode struct {
	AccountID    string
	Name         string
	Rating       Rating
	Weight       float64 // calculated rater weight
	PortfolioXLM float64
	Connections  int
	OwnScore     float64 // their own reputation score
	Distance     int     // 1 = direct rater, 2 = rater of rater
}

// Graph represents a 2-level reputation graph for an account.
type Graph struct {
	TargetAccountID string
	TargetName      string
	Score           *Score
	Level1Nodes     []GraphNode // Direct raters (gave A/B/C/D to target)
	Level2Nodes     []GraphNode // Raters of the level 1 raters
}

// RatingToValue converts a rating letter to numeric value.
// Deprecated: Use Rating.Value() method instead.
func RatingToValue(rating Rating) float64 {
	return rating.Value()
}
