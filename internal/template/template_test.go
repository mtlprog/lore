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
	slice := funcMap["slice"].(func(string, ...int) string)

	tests := []struct {
		name     string
		s        string
		indices  []int
		expected string
	}{
		// No indices (returns whole string)
		{"no indices", "hello", []int{}, "hello"},
		// Single index (start only)
		{"slice from middle", "hello world", []int{6}, "world"},
		{"slice from start", "hello", []int{0}, "hello"},
		{"slice from end", "hello", []int{5}, ""},
		{"negative start clamped", "hello", []int{-1}, "hello"},
		{"start beyond length", "hello", []int{10}, ""},
		// Two indices (start, end)
		{"slice with end", "hello", []int{0, 1}, "h"},
		{"slice middle portion", "hello world", []int{6, 11}, "world"},
		{"slice first char (NRX case)", "NRX", []int{0, 1}, "N"},
		{"end beyond length clamped", "hello", []int{0, 100}, "hello"},
		{"start equals end", "hello", []int{2, 2}, ""},
		{"start greater than end", "hello", []int{3, 2}, ""},
		// Empty string
		{"empty string no indices", "", []int{}, ""},
		{"empty string with indices", "", []int{0, 1}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slice(tt.s, tt.indices...)
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

func TestContainsTag(t *testing.T) {
	containsTag := funcMap["containsTag"].(func([]string, string) bool)

	tests := []struct {
		name     string
		tags     []string
		tag      string
		expected bool
	}{
		{"tag present", []string{"Belgrade", "Programmer"}, "Belgrade", true},
		{"tag not present", []string{"Belgrade", "Programmer"}, "Developer", false},
		{"empty tags", []string{}, "Belgrade", false},
		{"nil tags", nil, "Belgrade", false},
		{"empty tag search", []string{"Belgrade"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsTag(tt.tags, tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTagURL(t *testing.T) {
	tagURL := funcMap["tagURL"].(func([]string, string, bool, string) string)

	tests := []struct {
		name         string
		currentTags  []string
		tag          string
		add          bool
		currentQuery string
		expected     string
	}{
		{
			name:         "add tag to empty list",
			currentTags:  []string{},
			tag:          "Belgrade",
			add:          true,
			currentQuery: "",
			expected:     "/search?tag=Belgrade",
		},
		{
			name:         "add tag to existing list",
			currentTags:  []string{"Belgrade"},
			tag:          "Programmer",
			add:          true,
			currentQuery: "",
			expected:     "/search?tag=Belgrade&tag=Programmer",
		},
		{
			name:         "add duplicate tag (no change)",
			currentTags:  []string{"Belgrade"},
			tag:          "Belgrade",
			add:          true,
			currentQuery: "",
			expected:     "/search?tag=Belgrade",
		},
		{
			name:         "remove tag leaving others",
			currentTags:  []string{"Belgrade", "Programmer"},
			tag:          "Belgrade",
			add:          false,
			currentQuery: "",
			expected:     "/search?tag=Programmer",
		},
		{
			name:         "remove last tag",
			currentTags:  []string{"Belgrade"},
			tag:          "Belgrade",
			add:          false,
			currentQuery: "",
			expected:     "/search",
		},
		{
			name:         "remove tag not in list",
			currentTags:  []string{"Belgrade"},
			tag:          "Programmer",
			add:          false,
			currentQuery: "",
			expected:     "/search?tag=Belgrade",
		},
		{
			name:         "URL encodes special characters",
			currentTags:  []string{},
			tag:          "Tag With Spaces",
			add:          true,
			currentQuery: "",
			expected:     "/search?tag=Tag+With+Spaces",
		},
		{
			name:         "preserves query when adding tag",
			currentTags:  []string{},
			tag:          "Belgrade",
			add:          true,
			currentQuery: "test",
			expected:     "/search?q=test&tag=Belgrade",
		},
		{
			name:         "preserves query when removing last tag",
			currentTags:  []string{"Belgrade"},
			tag:          "Belgrade",
			add:          false,
			currentQuery: "test",
			expected:     "/search?q=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagURL(tt.currentTags, tt.tag, tt.add, tt.currentQuery)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("successful parsing", func(t *testing.T) {
		tmpl, err := New()
		require.NoError(t, err)
		require.NotNil(t, tmpl)

		// Verify all pages are parsed
		assert.Contains(t, tmpl.pages, "home.html")
		assert.Contains(t, tmpl.pages, "account.html")
		assert.Contains(t, tmpl.pages, "transaction.html")
		assert.Contains(t, tmpl.pages, "search.html")
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
				Tags       []string
				Trustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
				NFTTrustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
				LPShares []struct {
					PoolID       string
					ShareBalance string
					SharePercent string
					ReserveA     struct {
						AssetCode   string
						AssetIssuer string
						Amount      string
					}
					ReserveB struct {
						AssetCode   string
						AssetIssuer string
						Amount      string
					}
					XLMValue float64
				}
				Categories []struct {
					Name          string
					Color         string
					Relationships []struct {
						Type        string
						TargetID    string
						TargetName  string
						Direction   string
						IsMutual    bool
						IsConfirmed bool
					}
					IsEmpty bool
				}
				TrustRating *struct {
					CountA  int
					CountB  int
					CountC  int
					CountD  int
					Total   int
					Score   float64
					Grade   string
					Percent int
				}
				TotalXLMValue float64
				IsCorporate   bool
			}
			Operations *struct {
				Operations []struct {
					ID              string
					Type            string
					TypeDisplay     string
					TypeCategory    string
					CreatedAt       string
					TransactionHash string
				}
				NextCursor string
				HasMore    bool
			}
			AccountNames    map[string]string
			ReputationScore *struct {
				WeightedScore float64
				BaseScore     float64
				Grade         string
				RatingCountA  int
				RatingCountB  int
				RatingCountC  int
				RatingCountD  int
				TotalRatings  int
				TotalWeight   float64
			}
		}{
			Account: struct {
				ID         string
				Name       string
				About      string
				Websites   []string
				Tags       []string
				Trustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
				NFTTrustlines []struct {
					AssetCode   string
					AssetIssuer string
					Balance     string
					Limit       string
				}
				LPShares []struct {
					PoolID       string
					ShareBalance string
					SharePercent string
					ReserveA     struct {
						AssetCode   string
						AssetIssuer string
						Amount      string
					}
					ReserveB struct {
						AssetCode   string
						AssetIssuer string
						Amount      string
					}
					XLMValue float64
				}
				Categories []struct {
					Name          string
					Color         string
					Relationships []struct {
						Type        string
						TargetID    string
						TargetName  string
						Direction   string
						IsMutual    bool
						IsConfirmed bool
					}
					IsEmpty bool
				}
				TrustRating *struct {
					CountA  int
					CountB  int
					CountC  int
					CountD  int
					Total   int
					Score   float64
					Grade   string
					Percent int
				}
				TotalXLMValue float64
				IsCorporate   bool
			}{
				ID:       "GTEST1234567890",
				Name:     "Test Account",
				About:    "This is a test account",
				Websites: []string{"https://example.com"},
				Tags:     []string{"Belgrade", "Programmer"},
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
				NFTTrustlines: nil,
				LPShares:      nil,
				Categories:    nil,
				TrustRating:   nil,
				TotalXLMValue: 0,
				IsCorporate:   false,
			},
			Operations:      nil,
			AccountNames:    nil,
			ReputationScore: nil,
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

	t.Run("transaction template renders successfully", func(t *testing.T) {
		var buf bytes.Buffer
		data := struct {
			Transaction struct {
				Hash           string
				Successful     bool
				Ledger         int
				CreatedAt      string
				SourceAccount  string
				FeeCharged     string
				MemoType       string
				Memo           string
				OperationCount int
				Operations     []struct {
					ID              string
					Type            string
					TypeDisplay     string
					TypeCategory    string
					CreatedAt       string
					TransactionHash string
					Amount          string
					AssetCode       string
					From            string
					To              string
				}
			}
			AccountNames map[string]string
		}{
			Transaction: struct {
				Hash           string
				Successful     bool
				Ledger         int
				CreatedAt      string
				SourceAccount  string
				FeeCharged     string
				MemoType       string
				Memo           string
				OperationCount int
				Operations     []struct {
					ID              string
					Type            string
					TypeDisplay     string
					TypeCategory    string
					CreatedAt       string
					TransactionHash string
					Amount          string
					AssetCode       string
					From            string
					To              string
				}
			}{
				Hash:           "abc123def456",
				Successful:     true,
				Ledger:         12345678,
				CreatedAt:      "2024-01-29 12:00:00",
				SourceAccount:  "GTEST1234567890",
				FeeCharged:     "0.0001000",
				MemoType:       "none",
				Memo:           "",
				OperationCount: 1,
				Operations: []struct {
					ID              string
					Type            string
					TypeDisplay     string
					TypeCategory    string
					CreatedAt       string
					TransactionHash string
					Amount          string
					AssetCode       string
					From            string
					To              string
				}{
					{
						ID:              "op123",
						Type:            "payment",
						TypeDisplay:     "Payment",
						TypeCategory:    "payment",
						CreatedAt:       "2024-01-29 12:00:00",
						TransactionHash: "abc123def456",
						Amount:          "100.00",
						AssetCode:       "XLM",
						From:            "GFROM123",
						To:              "GTO456",
					},
				},
			},
			AccountNames: map[string]string{},
		}

		err := tmpl.Render(&buf, "transaction.html", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "TRANSACTION")
		assert.Contains(t, output, "abc123def456")
		assert.Contains(t, output, "SUCCESS")
		assert.Contains(t, output, "12345678")
	})

	t.Run("search template renders with query and tags", func(t *testing.T) {
		var buf bytes.Buffer
		data := struct {
			Query        string
			QueryTooLong bool
			Tags         []string
			AllTags      []struct {
				TagName string
				Count   int
			}
			Accounts   []any
			TotalCount int
			Offset     int
			NextOffset int
			HasMore    bool
			SortBy     string
		}{
			Query:        "test query",
			QueryTooLong: false,
			Tags:         []string{"Belgrade", "Programmer"},
			AllTags: []struct {
				TagName string
				Count   int
			}{
				{TagName: "Belgrade", Count: 10},
				{TagName: "Programmer", Count: 5},
			},
			Accounts:   []any{},
			TotalCount: 15,
			Offset:     0,
			NextOffset: 50,
			HasMore:    false,
			SortBy:     "balance",
		}

		err := tmpl.Render(&buf, "search.html", data)
		require.NoError(t, err)

		output := buf.String()
		// Verify template renders with query
		assert.Contains(t, output, "test query")
		// Verify meta description uses single quotes (not double quotes that would break HTML)
		assert.Contains(t, output, `Results for 'test query'`)
		// Verify the HTML is well-formed (meta content attribute is properly closed)
		assert.Contains(t, output, `<meta name="description" content="Search Montelibero accounts on Stellar blockchain. Results for 'test query' - 15 accounts found.">`)
	})

	t.Run("search template renders without query (tags only)", func(t *testing.T) {
		var buf bytes.Buffer
		data := struct {
			Query        string
			QueryTooLong bool
			Tags         []string
			AllTags      []struct {
				TagName string
				Count   int
			}
			Accounts   []any
			TotalCount int
			Offset     int
			NextOffset int
			HasMore    bool
			SortBy     string
		}{
			Query:        "",
			QueryTooLong: false,
			Tags:         []string{"Christian"},
			AllTags: []struct {
				TagName string
				Count   int
			}{
				{TagName: "Christian", Count: 1},
			},
			Accounts:   []any{},
			TotalCount: 1,
			Offset:     0,
			NextOffset: 50,
			HasMore:    false,
			SortBy:     "balance",
		}

		err := tmpl.Render(&buf, "search.html", data)
		require.NoError(t, err)

		output := buf.String()
		// Verify default meta description is used when no query
		assert.Contains(t, output, `Find participants and organizations by name, account ID, or tags.`)
	})
}
