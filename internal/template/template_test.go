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
			Companies           []any
			PersonsOffset       int
			CompaniesOffset     int
			NextPersonsOffset   int
			NextCompaniesOffset int
			HasMorePersons      bool
			HasMoreCompanies    bool
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
			Persons:           []any{},
			Companies:         []any{},
			HasMorePersons:    false,
			HasMoreCompanies:  false,
		}

		err := tmpl.Render(&buf, "home.html", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "LORE")
		assert.Contains(t, output, "100")  // TotalAccounts
		assert.Contains(t, output, "50")   // TotalPersons
		assert.Contains(t, output, "25")   // TotalCompanies
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
