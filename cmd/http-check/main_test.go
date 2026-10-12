package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "no headers",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name:     "trims whitespace",
			input:    []string{"  X-Test :  foo  "},
			expected: map[string]string{"X-Test": "foo"},
		},
		{
			name:     "value containing colons",
			input:    []string{"Location: https://example.com:8443/new"},
			expected: map[string]string{"Location": "https://example.com:8443/new"},
		},
		{
			name:     "multiple headers",
			input:    []string{"X-A: 1", "X-B: 2"},
			expected: map[string]string{"X-A": "1", "X-B": "2"},
		},
		{
			name:    "missing colon",
			input:   []string{"X-Test foo"},
			wantErr: true,
		},
		{
			name:    "missing colon with colon in value",
			input:   []string{"Location https://example.com"},
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   []string{": foo"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers, err := parseHeaders(test.input)
			if test.wantErr {
				assert.Error(t, err, "expected parse error for %q", test.input)
				return
			}

			assert.NoError(t, err, "unexpected parse error for %q", test.input)
			assert.Equal(t, test.expected, headers, "parsed headers do not match for %q", test.input)
		})
	}
}

func TestParseHeaderRegexes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "valid regex containing colons",
			input:    []string{"Location: ^https://example\\.com/"},
			expected: map[string]string{"Location": "^https://example\\.com/"},
		},
		{
			name:    "invalid regex",
			input:   []string{"Location: [0-9]++"},
			wantErr: true,
		},
		{
			name:    "invalid header format",
			input:   []string{"Location ^https://"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers, err := parseHeaderRegexes(test.input)
			if test.wantErr {
				assert.Error(t, err, "expected parse error for %q", test.input)
				return
			}

			assert.NoError(t, err, "unexpected parse error for %q", test.input)
			assert.Equal(t, test.expected, headers, "parsed headers do not match for %q", test.input)
		})
	}
}
