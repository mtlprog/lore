package reputation

import "testing"

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		// Pure ratings should return their grade
		{"pure A rating (4.0)", 4.0, "A"},
		{"pure B rating (3.0)", 3.0, "B"},
		{"pure C rating (2.0)", 2.0, "C"},
		{"pure D rating (1.0)", 1.0, "D"},

		// Boundary tests - grades align with rating values
		{"A boundary at 3.5", 3.5, "A"},
		{"B boundary just below 3.5", 3.49, "B"},
		{"B boundary at 2.5", 2.5, "B"},
		{"C boundary just below 2.5", 2.49, "C"},
		{"C boundary at 1.5", 1.5, "C"},
		{"D boundary just below 1.5", 1.49, "D"},
		{"D at small positive", 0.01, "D"},

		// Edge cases
		{"N/A at zero", 0, "N/A"},
		{"N/A at negative", -1.0, "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreToGrade(tt.score)
			if got != tt.want {
				t.Errorf("ScoreToGrade(%v) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestScore_Grade(t *testing.T) {
	s := &Score{WeightedScore: 3.75}
	if got := s.Grade(); got != "A" {
		t.Errorf("Score.Grade() = %q, want %q", got, "A")
	}
}

func TestRating_Value(t *testing.T) {
	tests := []struct {
		name   string
		rating Rating
		want   float64
	}{
		{"A rating", RatingA, 4.0},
		{"B rating", RatingB, 3.0},
		{"C rating", RatingC, 2.0},
		{"D rating", RatingD, 1.0},
		{"empty string", Rating(""), 0.0},
		{"invalid X", Rating("X"), 0.0},
		{"lowercase a", Rating("a"), 0.0},
		{"whitespace A", Rating("A "), 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rating.Value()
			if got != tt.want {
				t.Errorf("Rating(%q).Value() = %v, want %v", tt.rating, got, tt.want)
			}
		})
	}
}

func TestRating_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		rating Rating
		want   bool
	}{
		{"A is valid", RatingA, true},
		{"B is valid", RatingB, true},
		{"C is valid", RatingC, true},
		{"D is valid", RatingD, true},
		{"empty is invalid", Rating(""), false},
		{"X is invalid", Rating("X"), false},
		{"lowercase a is invalid", Rating("a"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rating.IsValid()
			if got != tt.want {
				t.Errorf("Rating(%q).IsValid() = %v, want %v", tt.rating, got, tt.want)
			}
		})
	}
}

func TestRatingToValue(t *testing.T) {
	// Test that RatingToValue (deprecated) still works
	if got := RatingToValue(RatingA); got != 4.0 {
		t.Errorf("RatingToValue(RatingA) = %v, want 4.0", got)
	}
}
