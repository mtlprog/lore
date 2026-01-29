package template

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	add := funcMap["add"].(func(int, int) int)

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 2, 3, 5},
		{"zero and positive", 0, 5, 5},
		{"negative numbers", -2, -3, -5},
		{"mixed signs", -2, 5, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := add(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	truncate := funcMap["truncate"].(func(string, int) string)

	tests := []struct {
		name     string
		s        string
		n        int
		expected string
	}{
		{"truncate long string", "hello world", 5, "hello"},
		{"string shorter than n", "hi", 10, "hi"},
		{"exact length", "hello", 5, "hello"},
		{"zero length", "hello", 0, ""},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.s, tt.n)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlice(t *testing.T) {
	slice := funcMap["slice"].(func(string, int) string)

	tests := []struct {
		name     string
		s        string
		start    int
		expected string
	}{
		{"slice from middle", "hello world", 6, "world"},
		{"slice from start", "hello", 0, "hello"},
		{"slice from end", "hello", 5, ""},
		{"negative start", "hello", -1, ""},
		{"start beyond length", "hello", 10, ""},
		{"empty string", "", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slice(tt.s, tt.start)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatNumber(t *testing.T) {
	formatNumber := funcMap["formatNumber"].(func(float64) string)

	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"small number", 123.0, "123"},
		{"thousands", 1234.0, "1 234"},
		{"millions", 3311282.0, "3 311 282"},
		{"zero", 0.0, "0"},
		{"negative", -1234.0, "-1 234"},
		{"rounding up", 1234.6, "1 235"},
		{"rounding down", 1234.4, "1 234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVotePower(t *testing.T) {
	votePower := funcMap["votePower"].(func(float64) int)

	tests := []struct {
		name       string
		totalVotes float64
		expected   int
	}{
		// Edge cases
		{"zero votes", 0.0, 0},
		{"negative votes", -5.0, 0},

		// 1-10 total votes = 1 vote power
		{"1 vote", 1.0, 1},
		{"5 votes", 5.0, 1},
		{"10 votes", 10.0, 1},

		// 11-100 total votes = 2 vote power
		{"11 votes", 11.0, 2},
		{"13 votes (Stanislav case)", 13.0, 2},
		{"50 votes", 50.0, 2},
		{"100 votes", 100.0, 2},

		// 101-1000 total votes = 3 vote power
		{"101 votes", 101.0, 3},
		{"500 votes", 500.0, 3},
		{"1000 votes", 1000.0, 3},

		// 1001-10000 total votes = 4 vote power
		{"1001 votes", 1001.0, 4},
		{"10000 votes", 10000.0, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := votePower(tt.totalVotes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddFloat(t *testing.T) {
	addFloat := funcMap["addFloat"].(func(float64, int) float64)

	tests := []struct {
		name     string
		a        float64
		b        int
		expected float64
	}{
		{"5.0 + 8", 5.0, 8, 13.0},
		{"0.0 + 0", 0.0, 0, 0.0},
		{"3.5 + 10", 3.5, 10, 13.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addFloat(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("successful parsing", func(t *testing.T) {
		tmpl, err := New()
		require.NoError(t, err)
		require.NotNil(t, tmpl)

		// Verify both pages are parsed
		assert.Contains(t, tmpl.pages, "home.html")
		assert.Contains(t, tmpl.pages, "account.html")
	})
}

func TestRender(t *testing.T) {
	tmpl, err := New()
	require.NoError(t, err)

	t.Run("unknown template returns error", func(t *testing.T) {
		var buf bytes.Buffer
		err := tmpl.Render(&buf, "nonexistent.html", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template nonexistent.html not found")
	})

	t.Run("home template renders successfully", func(t *testing.T) {
		var buf bytes.Buffer
		data := struct {
			Stats struct {
				TotalAccounts  int
				TotalPersons   int
				TotalCompanies int
				TotalXLMValue  float64
			}
			Persons             []any
			Corporate           []any
			PersonsOffset       int
			CorporateOffset     int
			NextPersonsOffset   int
			NextCorporateOffset int
			HasMorePersons      bool
			HasMoreCorporate    bool
		}{
			Stats: struct {
				TotalAccounts  int
				TotalPersons   int
				TotalCompanies int
				TotalXLMValue  float64
			}{
				TotalAccounts:  100,
				TotalPersons:   50,
				TotalCompanies: 25,
				TotalXLMValue:  1000000.0,
			},
			Persons:          []any{},
			Corporate:        []any{},
			HasMorePersons:   false,
			HasMoreCorporate: false,
		}

		err := tmpl.Render(&buf, "home.html", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "LORE")
		assert.Contains(t, output, "100") // TotalAccounts
		assert.Contains(t, output, "50")  // TotalPersons
		assert.Contains(t, output, "25")  // TotalCompanies
	})

	t.Run("account template renders successfully", func(t *testing.T) {
		var buf bytes.Buffer
		data := struct {
			Account struct {
				ID         string
				Name       string
				About      string
				Websites   []string
				Trustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
			}
		}{
			Account: struct {
				ID         string
				Name       string
				About      string
				Websites   []string
				Trustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
			}{
				ID:       "GTEST1234567890",
				Name:     "Test Account",
				About:    "This is a test account",
				Websites: []string{"https://example.com"},
				Trustlines: []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}{
					{
						AssetCode:   "MTLAP",
						AssetIssuer: "GISSUER",
						Balance:     "100.00",
						Limit:       "1000.00",
					},
				},
			},
		}

		err := tmpl.Render(&buf, "account.html", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Test Account")
		assert.Contains(t, output, "GTEST1234567890")
		assert.Contains(t, output, "This is a test account")
		assert.Contains(t, output, "https://example.com")
		assert.Contains(t, output, "MTLAP")
	})
}
