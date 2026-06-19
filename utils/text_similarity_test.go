package utils_test

import (
	"testing"

	"my-app/utils"
)

func TestTextSimilarity(t *testing.T) {
	tests := []struct {
		a, b     string
		minScore float32
	}{
		{"hello world", "hello world", 1.0},
		{"Hello World!", "hello   world", 0.9},
		{"oauth redirect handling for github", "GitHub OAuth redirect URI setup", 0.3},
		{"exact duplicate text here", "exact duplicate text here", 1.0},
		{"", "anything", 0},
		{"foo", "bar", 0},
	}

	for _, tt := range tests {
		got := utils.TextSimilarity(tt.a, tt.b)
		if tt.minScore == 0 {
			if got != 0 {
				t.Errorf("TextSimilarity(%q, %q) = %v, want 0", tt.a, tt.b, got)
			}
			continue
		}
		if got < tt.minScore {
			t.Errorf("TextSimilarity(%q, %q) = %v, want >= %v", tt.a, tt.b, got, tt.minScore)
		}
	}
}
