package sync

import (
	"encoding/base64"
	"testing"

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
