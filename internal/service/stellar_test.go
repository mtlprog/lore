package service

import (
	"encoding/base64"
	"testing"

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
