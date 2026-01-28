package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAssociationTags(t *testing.T) {
	validAccountID := "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"

	tests := []struct {
		name     string
		rawData  map[string]string
		expected []AssociationTag
	}{
		{
			name:     "empty data",
			rawData:  map[string]string{},
			expected: nil,
		},
		{
			name: "valid Program tag",
			rawData: map[string]string{
				"Program" + validAccountID: "1",
			},
			expected: []AssociationTag{
				{TagName: "Program", TagIndex: "0", TargetAccountID: validAccountID},
			},
		},
		{
			name: "valid Faction tag",
			rawData: map[string]string{
				"Faction" + validAccountID: "1",
			},
			expected: []AssociationTag{
				{TagName: "Faction", TagIndex: "0", TargetAccountID: validAccountID},
			},
		},
		{
			name: "tag with index",
			rawData: map[string]string{
				"Program" + validAccountID + "1": "1",
			},
			expected: []AssociationTag{
				{TagName: "Program", TagIndex: "1", TargetAccountID: validAccountID},
			},
		},
		{
			name: "multiple tags",
			rawData: map[string]string{
				"Program" + validAccountID:       "1",
				"Faction" + validAccountID + "2": "1",
			},
			expected: []AssociationTag{
				{TagName: "Program", TagIndex: "0", TargetAccountID: validAccountID},
				{TagName: "Faction", TagIndex: "2", TargetAccountID: validAccountID},
			},
		},
		{
			name: "invalid - unknown tag type",
			rawData: map[string]string{
				"Unknown" + validAccountID: "1",
			},
			expected: nil,
		},
		{
			name: "invalid - key too short",
			rawData: map[string]string{
				"ProgramGCNVDZ": "1",
			},
			expected: nil,
		},
		{
			name: "invalid - account ID not starting with G",
			rawData: map[string]string{
				"ProgramACNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA": "1",
			},
			expected: nil,
		},
		{
			name: "ignores non-tag keys",
			rawData: map[string]string{
				"Name":    "Test",
				"Website": "https://example.com",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAssociationTags(tt.rawData)

			if tt.expected == nil {
				assert.Empty(t, result)
				return
			}

			// Sort both slices for comparison (order may vary due to map iteration)
			assert.Len(t, result, len(tt.expected))

			// Check each expected tag is present
			for _, exp := range tt.expected {
				found := false
				for _, res := range result {
					if res.TagName == exp.TagName && res.TagIndex == exp.TagIndex && res.TargetAccountID == exp.TargetAccountID {
						found = true
						break
					}
				}
				assert.True(t, found, "expected tag not found: %+v", exp)
			}
		})
	}
}
