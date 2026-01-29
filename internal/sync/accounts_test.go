package sync

import (
	"encoding/base64"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stretchr/testify/assert"
)

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid base64",
			input:    base64.StdEncoding.EncodeToString([]byte("Hello World")),
			expected: "Hello World",
		},
		{
			name:     "base64 with whitespace",
			input:    base64.StdEncoding.EncodeToString([]byte("  trimmed  ")),
			expected: "trimmed",
		},
		{
			name:     "invalid base64",
			input:    "not-valid-base64!!!",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeBase64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseNumberedKey(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedBase  string
		expectedIndex string
	}{
		{
			name:          "key without number",
			key:           "Name",
			expectedBase:  "Name",
			expectedIndex: "",
		},
		{
			name:          "key with zero",
			key:           "Website0",
			expectedBase:  "Website",
			expectedIndex: "0",
		},
		{
			name:          "key with number",
			key:           "Website1",
			expectedBase:  "Website",
			expectedIndex: "1",
		},
		{
			name:          "key with large number",
			key:           "Website123",
			expectedBase:  "Website",
			expectedIndex: "123",
		},
		{
			name:          "key with leading zeros",
			key:           "PartOf002",
			expectedBase:  "PartOf",
			expectedIndex: "002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, index := parseNumberedKey(tt.key)
			assert.Equal(t, tt.expectedBase, base)
			assert.Equal(t, tt.expectedIndex, index)
		})
	}
}

func TestParseRelationship(t *testing.T) {
	validAccountID := "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"

	tests := []struct {
		name     string
		key      string
		value    string
		expected *Relationship
	}{
		{
			name:  "valid PartOf relationship",
			key:   "PartOf" + validAccountID,
			value: "1",
			expected: &Relationship{
				TargetAccountID: validAccountID,
				RelationType:    RelationPartOf,
				RelationIndex:   0,
			},
		},
		{
			name:  "valid relationship with index",
			key:   "MyPart" + validAccountID + "1",
			value: "1",
			expected: &Relationship{
				TargetAccountID: validAccountID,
				RelationType:    RelationMyPart,
				RelationIndex:   1,
			},
		},
		{
			name:     "invalid - short account ID",
			key:      "PartOfGCNVDZ",
			value:    "1",
			expected: nil,
		},
		{
			name:     "invalid - unknown relation type",
			key:      "Unknown" + validAccountID,
			value:    "1",
			expected: nil,
		},
		{
			name:     "invalid - not a relationship key",
			key:      "Name",
			value:    "test",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRelationship(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseManageData(t *testing.T) {
	validAccountID := "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"

	tests := []struct {
		name                    string
		data                    map[string]string
		expectedMeta            int
		expectedRels            int
		expectedDelegate        *string
		expectedCouncilDelegate *string
		expectedCouncil         bool
	}{
		{
			name: "basic metadata",
			data: map[string]string{
				"Name":  base64.StdEncoding.EncodeToString([]byte("Test Account")),
				"About": base64.StdEncoding.EncodeToString([]byte("Description")),
			},
			expectedMeta:            2,
			expectedRels:            0,
			expectedDelegate:        nil,
			expectedCouncilDelegate: nil,
			expectedCouncil:         false,
		},
		{
			name: "with delegate",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
			},
			expectedMeta:            0,
			expectedRels:            0,
			expectedDelegate:        &validAccountID,
			expectedCouncilDelegate: nil,
			expectedCouncil:         false,
		},
		{
			name: "council ready with mtla_c_delegate = ready",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte("ready")),
			},
			expectedMeta:            0,
			expectedRels:            0,
			expectedDelegate:        nil,
			expectedCouncilDelegate: nil,
			expectedCouncil:         true,
		},
		{
			name: "council delegation with mtla_c_delegate = account ID",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
			},
			expectedMeta:            0,
			expectedRels:            0,
			expectedDelegate:        nil,
			expectedCouncilDelegate: &validAccountID,
			expectedCouncil:         false,
		},
		{
			name: "with relationship",
			data: map[string]string{
				"PartOf" + validAccountID: base64.StdEncoding.EncodeToString([]byte("1")),
			},
			expectedMeta:            0,
			expectedRels:            1,
			expectedDelegate:        nil,
			expectedCouncilDelegate: nil,
			expectedCouncil:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, rels, delegate, councilDelegate, council := parseManageData(tt.data)
			assert.Len(t, meta, tt.expectedMeta)
			assert.Len(t, rels, tt.expectedRels)
			assert.Equal(t, tt.expectedDelegate, delegate)
			assert.Equal(t, tt.expectedCouncilDelegate, councilDelegate)
			assert.Equal(t, tt.expectedCouncil, council)
		})
	}
}

func TestParseAccountData(t *testing.T) {
	tests := []struct {
		name           string
		account        *horizon.Account
		expectedID     string
		expectedBalLen int
		hasNativeXLM   bool
		hasCreditAsset bool
	}{
		{
			name: "account with only native XLM",
			account: &horizon.Account{
				ID: "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
				Balances: []horizon.Balance{
					{
						Balance: "100.0000000",
						Asset:   base.Asset{Type: "native"},
					},
				},
			},
			expectedID:     "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expectedBalLen: 1,
			hasNativeXLM:   true,
			hasCreditAsset: false,
		},
		{
			name: "account with credit asset",
			account: &horizon.Account{
				ID: "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
				Balances: []horizon.Balance{
					{
						Balance: "50.0000000",
						Asset: base.Asset{
							Type:   "credit_alphanum4",
							Code:   "USDC",
							Issuer: "GISSUER123456789012345678901234567890123456789012345",
						},
					},
				},
			},
			expectedID:     "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expectedBalLen: 1,
			hasNativeXLM:   false,
			hasCreditAsset: true,
		},
		{
			name: "account with native and credit",
			account: &horizon.Account{
				ID: "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
				Balances: []horizon.Balance{
					{
						Balance: "100.0000000",
						Asset:   base.Asset{Type: "native"},
					},
					{
						Balance: "50.0000000",
						Asset: base.Asset{
							Type:   "credit_alphanum4",
							Code:   "USDC",
							Issuer: "GISSUER123456789012345678901234567890123456789012345",
						},
					},
				},
			},
			expectedID:     "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expectedBalLen: 2,
			hasNativeXLM:   true,
			hasCreditAsset: true,
		},
		{
			name: "account with empty balances",
			account: &horizon.Account{
				ID:       "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
				Balances: []horizon.Balance{},
			},
			expectedID:     "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expectedBalLen: 0,
			hasNativeXLM:   false,
			hasCreditAsset: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAccountData(tt.account)

			assert.Equal(t, tt.expectedID, result.ID)
			assert.Len(t, result.Balances, tt.expectedBalLen)

			// Check for native XLM
			hasXLM := false
			for _, bal := range result.Balances {
				if bal.AssetCode == "XLM" && bal.AssetIssuer == "" {
					hasXLM = true
					break
				}
			}
			assert.Equal(t, tt.hasNativeXLM, hasXLM)

			// Check for credit asset
			hasCredit := false
			for _, bal := range result.Balances {
				if bal.AssetCode != "XLM" && bal.AssetIssuer != "" {
					hasCredit = true
					break
				}
			}
			assert.Equal(t, tt.hasCreditAsset, hasCredit)
		})
	}
}

func TestFindBalance(t *testing.T) {
	balances := []Balance{
		{AssetCode: "XLM", AssetIssuer: "", Balance: decimal.RequireFromString("100.0000000")},
		{AssetCode: "USDC", AssetIssuer: "GISSUER1", Balance: decimal.RequireFromString("50.0000000")},
		{AssetCode: "MTLAP", AssetIssuer: "GISSUER2", Balance: decimal.RequireFromString("5.0000000")},
	}

	tests := []struct {
		name     string
		code     string
		issuer   string
		expected decimal.Decimal
	}{
		{
			name:     "find native XLM",
			code:     "XLM",
			issuer:   "",
			expected: decimal.RequireFromString("100.0000000"),
		},
		{
			name:     "find USDC",
			code:     "USDC",
			issuer:   "GISSUER1",
			expected: decimal.RequireFromString("50.0000000"),
		},
		{
			name:     "find MTLAP",
			code:     "MTLAP",
			issuer:   "GISSUER2",
			expected: decimal.RequireFromString("5.0000000"),
		},
		{
			name:     "asset not found",
			code:     "UNKNOWN",
			issuer:   "GISSUER",
			expected: decimal.Zero,
		},
		{
			name:     "wrong issuer",
			code:     "USDC",
			issuer:   "WRONG_ISSUER",
			expected: decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findBalance(balances, tt.code, tt.issuer)
			assert.True(t, tt.expected.Equal(result), "expected %s, got %s", tt.expected, result)
		})
	}
}

func TestFindBalanceEmptySlice(t *testing.T) {
	result := findBalance([]Balance{}, "XLM", "")
	assert.True(t, decimal.Zero.Equal(result))
}

func TestParseManageDataEdgeCases(t *testing.T) {
	validAccountID := "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"

	tests := []struct {
		name            string
		data            map[string]string
		expectedCouncil bool
	}{
		{
			name:            "empty data",
			data:            map[string]string{},
			expectedCouncil: false,
		},
		{
			name: "council ready with lowercase ready",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte("ready")),
			},
			expectedCouncil: true,
		},
		{
			name: "council ready with uppercase READY",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte("READY")),
			},
			expectedCouncil: true,
		},
		{
			name: "council ready with mixed case Ready",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte("Ready")),
			},
			expectedCouncil: true,
		},
		{
			name: "council delegate with account ID is not ready",
			data: map[string]string{
				"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
			},
			expectedCouncil: false,
		},
		{
			name: "delegate with invalid account ID (too short)",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte("GSHORT")),
			},
			expectedCouncil: false,
		},
		{
			name: "delegate with invalid account ID (wrong prefix)",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte("ACNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA")),
			},
			expectedCouncil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, delegate, _, council := parseManageData(tt.data)
			assert.Equal(t, tt.expectedCouncil, council)

			if tt.name == "delegate with invalid account ID (too short)" || tt.name == "delegate with invalid account ID (wrong prefix)" {
				assert.Nil(t, delegate)
			}
		})
	}

	t.Run("valid delegate", func(t *testing.T) {
		data := map[string]string{
			"mtla_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
		}
		_, _, delegate, _, _ := parseManageData(data)
		assert.NotNil(t, delegate)
		assert.Equal(t, validAccountID, *delegate)
	})

	t.Run("valid council delegate", func(t *testing.T) {
		data := map[string]string{
			"mtla_c_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
		}
		_, _, _, councilDelegate, council := parseManageData(data)
		assert.NotNil(t, councilDelegate)
		assert.Equal(t, validAccountID, *councilDelegate)
		assert.False(t, council) // Not council-ready, just delegating
	})
}

func TestParseManageDataNumberedKeysSorting(t *testing.T) {
	data := map[string]string{
		"Website2": base64.StdEncoding.EncodeToString([]byte("https://third.com")),
		"Website0": base64.StdEncoding.EncodeToString([]byte("https://first.com")),
		"Website1": base64.StdEncoding.EncodeToString([]byte("https://second.com")),
	}

	meta, _, _, _, _ := parseManageData(data)

	var websites []Metadata
	for _, m := range meta {
		if m.Key == "Website" {
			websites = append(websites, m)
		}
	}

	assert.Len(t, websites, 3)

	assert.Equal(t, "0", websites[0].Index)
	assert.Equal(t, "https://first.com", websites[0].Value)
	assert.Equal(t, "1", websites[1].Index)
	assert.Equal(t, "https://second.com", websites[1].Value)
	assert.Equal(t, "2", websites[2].Index)
	assert.Equal(t, "https://third.com", websites[2].Value)
}
