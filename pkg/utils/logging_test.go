package utils

import (
	"testing"
)

func TestObfuscateCredential(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long credential",
			input:    "AKIAIOSFODNN7EXAMPLE",
			expected: "AKIA************MPLE",
		},
		{
			name:     "exactly 9 characters",
			input:    "123456789",
			expected: "1234*6789",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "short credential (7 chars)",
			input:    "1234567",
			expected: "****",
		},
		{
			name:     "very short credential",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "single character",
			input:    "x",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ObfuscateCredential(tt.input)
			if result != tt.expected {
				t.Errorf("ObfuscateCredential(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
