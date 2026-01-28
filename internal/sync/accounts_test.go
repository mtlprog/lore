package sync

import (
	"encoding/base64"
	"testing"

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
			expectedIndex: "0",
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
				RelationType:    "PartOf",
				RelationIndex:   "0",
			},
		},
		{
			name:  "valid relationship with index",
			key:   "MyPart" + validAccountID + "1",
			value: "1",
			expected: &Relationship{
				TargetAccountID: validAccountID,
				RelationType:    "MyPart",
				RelationIndex:   "1",
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
		name             string
		data             map[string]string
		expectedMeta     int
		expectedRels     int
		expectedDelegate *string
		expectedCouncil  bool
	}{
		{
			name: "basic metadata",
			data: map[string]string{
				"Name":  base64.StdEncoding.EncodeToString([]byte("Test Account")),
				"About": base64.StdEncoding.EncodeToString([]byte("Description")),
			},
			expectedMeta:     2,
			expectedRels:     0,
			expectedDelegate: nil,
			expectedCouncil:  false,
		},
		{
			name: "with delegate",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
			},
			expectedMeta:     0,
			expectedRels:     0,
			expectedDelegate: &validAccountID,
			expectedCouncil:  false,
		},
		{
			name: "council ready true",
			data: map[string]string{
				"mtla_council_ready": base64.StdEncoding.EncodeToString([]byte("1")),
			},
			expectedMeta:     0,
			expectedRels:     0,
			expectedDelegate: nil,
			expectedCouncil:  true,
		},
		{
			name: "with relationship",
			data: map[string]string{
				"PartOf" + validAccountID: base64.StdEncoding.EncodeToString([]byte("1")),
			},
			expectedMeta:     0,
			expectedRels:     1,
			expectedDelegate: nil,
			expectedCouncil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, rels, delegate, council := parseManageData(tt.data)
			assert.Len(t, meta, tt.expectedMeta)
			assert.Len(t, rels, tt.expectedRels)
			assert.Equal(t, tt.expectedDelegate, delegate)
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
		{AssetCode: "XLM", AssetIssuer: "", Balance: "100.0000000"},
		{AssetCode: "USDC", AssetIssuer: "GISSUER1", Balance: "50.0000000"},
		{AssetCode: "MTLAP", AssetIssuer: "GISSUER2", Balance: "5.0000000"},
	}

	tests := []struct {
		name     string
		code     string
		issuer   string
		expected string
	}{
		{
			name:     "find native XLM",
			code:     "XLM",
			issuer:   "",
			expected: "100.0000000",
		},
		{
			name:     "find USDC",
			code:     "USDC",
			issuer:   "GISSUER1",
			expected: "50.0000000",
		},
		{
			name:     "find MTLAP",
			code:     "MTLAP",
			issuer:   "GISSUER2",
			expected: "5.0000000",
		},
		{
			name:     "asset not found",
			code:     "UNKNOWN",
			issuer:   "GISSUER",
			expected: "0",
		},
		{
			name:     "wrong issuer",
			code:     "USDC",
			issuer:   "WRONG_ISSUER",
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findBalance(balances, tt.code, tt.issuer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindBalanceEmptySlice(t *testing.T) {
	result := findBalance([]Balance{}, "XLM", "")
	assert.Equal(t, "0", result)
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
			name: "council ready with lowercase true",
			data: map[string]string{
				"mtla_council_ready": base64.StdEncoding.EncodeToString([]byte("true")),
			},
			expectedCouncil: true,
		},
		{
			name: "council ready with uppercase TRUE",
			data: map[string]string{
				"mtla_council_ready": base64.StdEncoding.EncodeToString([]byte("TRUE")),
			},
			expectedCouncil: true,
		},
		{
			name: "council ready with 0",
			data: map[string]string{
				"mtla_council_ready": base64.StdEncoding.EncodeToString([]byte("0")),
			},
			expectedCouncil: false,
		},
		{
			name: "council ready with false",
			data: map[string]string{
				"mtla_council_ready": base64.StdEncoding.EncodeToString([]byte("false")),
			},
			expectedCouncil: false,
		},
		{
			name: "delegate with invalid account ID (too short)",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte("GSHORT")),
			},
			expectedCouncil: false, // delegate should be nil
		},
		{
			name: "delegate with invalid account ID (wrong prefix)",
			data: map[string]string{
				"mtla_delegate": base64.StdEncoding.EncodeToString([]byte("ACNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA")),
			},
			expectedCouncil: false, // delegate should be nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, delegate, council := parseManageData(tt.data)
			assert.Equal(t, tt.expectedCouncil, council)

			// Check delegate validity for specific test cases
			if tt.name == "delegate with invalid account ID (too short)" || tt.name == "delegate with invalid account ID (wrong prefix)" {
				assert.Nil(t, delegate)
			}
		})
	}

	// Test valid delegate
	t.Run("valid delegate", func(t *testing.T) {
		data := map[string]string{
			"mtla_delegate": base64.StdEncoding.EncodeToString([]byte(validAccountID)),
		}
		_, _, delegate, _ := parseManageData(data)
		assert.NotNil(t, delegate)
		assert.Equal(t, validAccountID, *delegate)
	})
}

func TestParseManageDataNumberedKeysSorting(t *testing.T) {
	// Test that numbered keys are sorted correctly
	data := map[string]string{
		"Website2": base64.StdEncoding.EncodeToString([]byte("https://third.com")),
		"Website0": base64.StdEncoding.EncodeToString([]byte("https://first.com")),
		"Website1": base64.StdEncoding.EncodeToString([]byte("https://second.com")),
	}

	meta, _, _, _ := parseManageData(data)

	// Find website entries
	var websites []Metadata
	for _, m := range meta {
		if m.Key == "Website" {
			websites = append(websites, m)
		}
	}

	assert.Len(t, websites, 3)

	// Verify sorted order
	assert.Equal(t, "0", websites[0].Index)
	assert.Equal(t, "https://first.com", websites[0].Value)
	assert.Equal(t, "1", websites[1].Index)
	assert.Equal(t, "https://second.com", websites[1].Value)
	assert.Equal(t, "2", websites[2].Index)
	assert.Equal(t, "https://third.com", websites[2].Value)
}
