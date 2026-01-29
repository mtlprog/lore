package service

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"github.com/mtlprog/lore/internal/model"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stretchr/testify/assert"
)

func TestParseNumberedDataKeys(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		prefix   string
		expected []string
	}{
		{
			name:     "empty data",
			data:     map[string]string{},
			prefix:   "Website",
			expected: []string{},
		},
		{
			name: "single key without number",
			data: map[string]string{
				"Website": encode("https://example.com"),
			},
			prefix:   "Website",
			expected: []string{"https://example.com"},
		},
		{
			name: "numbered keys",
			data: map[string]string{
				"Website":  encode("https://first.com"),
				"Website1": encode("https://second.com"),
				"Website2": encode("https://third.com"),
			},
			prefix:   "Website",
			expected: []string{"https://first.com", "https://second.com", "https://third.com"},
		},
		{
			name: "numbered keys with leading zeros",
			data: map[string]string{
				"Website":     encode("https://zero.com"),
				"Website0001": encode("https://one.com"),
				"Website0002": encode("https://two.com"),
			},
			prefix:   "Website",
			expected: []string{"https://zero.com", "https://one.com", "https://two.com"},
		},
		{
			name: "out of order keys",
			data: map[string]string{
				"Website3": encode("https://three.com"),
				"Website1": encode("https://one.com"),
				"Website":  encode("https://zero.com"),
				"Website2": encode("https://two.com"),
			},
			prefix:   "Website",
			expected: []string{"https://zero.com", "https://one.com", "https://two.com", "https://three.com"},
		},
		{
			name: "mixed keys with different prefixes",
			data: map[string]string{
				"Website":  encode("https://web.com"),
				"Website1": encode("https://web1.com"),
				"Name":     encode("Test Name"),
				"About":    encode("Test About"),
			},
			prefix:   "Website",
			expected: []string{"https://web.com", "https://web1.com"},
		},
		{
			name: "skip empty values",
			data: map[string]string{
				"Website":  encode("https://valid.com"),
				"Website1": "",
				"Website2": encode("https://another.com"),
			},
			prefix:   "Website",
			expected: []string{"https://valid.com", "https://another.com"},
		},
		{
			name: "skip invalid base64",
			data: map[string]string{
				"Website":  encode("https://valid.com"),
				"Website1": "not-valid-base64!!!",
				"Website2": encode("https://another.com"),
			},
			prefix:   "Website",
			expected: []string{"https://valid.com", "https://another.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumberedDataKeys(tt.data, tt.prefix)
			if len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

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
			input:    encode("Hello, World!"),
			expected: "Hello, World!",
		},
		{
			name:     "invalid base64",
			input:    "not-valid!!!",
			expected: "",
		},
		{
			name:     "whitespace trimmed",
			input:    encode("  test  "),
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeBase64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestFindAssetBalance(t *testing.T) {
	balances := []horizon.Balance{
		{
			Balance: "100.0000000",
			Asset: base.Asset{
				Type:   "native",
				Code:   "",
				Issuer: "",
			},
		},
		{
			Balance: "50.0000000",
			Asset: base.Asset{
				Type:   "credit_alphanum4",
				Code:   "MTLAP",
				Issuer: "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			},
		},
		{
			Balance: "25.0000000",
			Asset: base.Asset{
				Type:   "credit_alphanum4",
				Code:   "USDC",
				Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			},
		},
	}

	tests := []struct {
		name     string
		code     string
		issuer   string
		expected string
	}{
		{
			name:     "find MTLAP",
			code:     "MTLAP",
			issuer:   "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA",
			expected: "50.0000000",
		},
		{
			name:     "find USDC",
			code:     "USDC",
			issuer:   "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
			expected: "25.0000000",
		},
		{
			name:     "asset not found",
			code:     "BTC",
			issuer:   "GISSUER",
			expected: "0",
		},
		{
			name:     "wrong issuer",
			code:     "MTLAP",
			issuer:   "GWRONGISSUER",
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAssetBalance(balances, tt.code, tt.issuer)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindAssetBalance_EmptyBalances(t *testing.T) {
	result := findAssetBalance([]horizon.Balance{}, "MTLAP", "GISSUER")
	assert.Equal(t, "0", result)
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "horizon 404 error",
			err: &horizonclient.Error{
				Response: &http.Response{
					StatusCode: 404,
				},
			},
			expected: true,
		},
		{
			name: "horizon 500 error",
			err: &horizonclient.Error{
				Response: &http.Response{
					StatusCode: 500,
				},
			},
			expected: false,
		},
		{
			name: "horizon error without response",
			err: &horizonclient.Error{
				Response: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStellarService(t *testing.T) {
	t.Run("creates service with custom URL", func(t *testing.T) {
		s := NewStellarService("https://custom-horizon.example.com")
		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
		assert.Equal(t, "https://custom-horizon.example.com", s.client.HorizonURL)
	})

	t.Run("creates service with default URL", func(t *testing.T) {
		s := NewStellarService("https://horizon.stellar.org")
		assert.NotNil(t, s)
		assert.Equal(t, "https://horizon.stellar.org", s.client.HorizonURL)
	})
}

func TestIsSpamOperation(t *testing.T) {
	tests := []struct {
		name string
		op   model.Operation
		want bool
	}{
		{
			name: "create_claimable_balance is spam",
			op:   model.Operation{Type: "create_claimable_balance"},
			want: true,
		},
		{
			name: "claim_claimable_balance is spam",
			op:   model.Operation{Type: "claim_claimable_balance"},
			want: true,
		},
		{
			name: "small XLM payment (0.5) is spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "0.5"},
			want: true,
		},
		{
			name: "very small XLM payment (0.0001) is spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "0.0001"},
			want: true,
		},
		{
			name: "exactly 1 XLM is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "1.0"},
			want: false,
		},
		{
			name: "exactly 1 XLM integer is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "1"},
			want: false,
		},
		{
			name: "large XLM payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "100.5"},
			want: false,
		},
		{
			name: "small non-XLM payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "USDC", Amount: "0.5"},
			want: false,
		},
		{
			name: "small MTLAP payment is NOT spam",
			op:   model.Operation{Type: "payment", AssetCode: "MTLAP", Amount: "0.001"},
			want: false,
		},
		{
			name: "manage_data is NOT spam",
			op:   model.Operation{Type: "manage_data", DataName: "test"},
			want: false,
		},
		{
			name: "change_trust is NOT spam",
			op:   model.Operation{Type: "change_trust", AssetCode: "MTLAP"},
			want: false,
		},
		{
			name: "invalid amount string is NOT spam (parse fails)",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: "invalid"},
			want: false,
		},
		{
			name: "empty amount is NOT spam (parse fails)",
			op:   model.Operation{Type: "payment", AssetCode: "XLM", Amount: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSpamOperation(tt.op)
			assert.Equal(t, tt.want, got)
		})
	}
}
