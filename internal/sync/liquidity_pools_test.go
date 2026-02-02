package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReserveAsset(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCode   string
		expectedIssuer string
	}{
		{
			name:           "native XLM",
			input:          "native",
			expectedCode:   "XLM",
			expectedIssuer: "",
		},
		{
			name:           "credit asset with issuer",
			input:          "USDC:GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			expectedCode:   "USDC",
			expectedIssuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
		},
		{
			name:           "MTL token",
			input:          "MTL:GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
			expectedCode:   "MTL",
			expectedIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
		},
		{
			name:           "single part no colon",
			input:          "MTLAP",
			expectedCode:   "MTLAP",
			expectedIssuer: "",
		},
		{
			name:           "empty string",
			input:          "",
			expectedCode:   "",
			expectedIssuer: "",
		},
		{
			name:           "long asset code",
			input:          "EURDEBTLOCK:GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
			expectedCode:   "EURDEBTLOCK",
			expectedIssuer: "GACKTN5DAZGWXRWB2WLM6OPBDHAMT6SJNGLJZPQMEZBUR4JUGBX2UK7V",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, issuer := parseReserveAsset(tt.input)
			assert.Equal(t, tt.expectedCode, code)
			assert.Equal(t, tt.expectedIssuer, issuer)
		})
	}
}
