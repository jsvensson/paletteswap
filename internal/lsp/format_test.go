package lsp

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic formatting",
			input:    `meta{name="Test"author="Author"}`,
			expected: `meta { name = "Test" author = "Author" }`,
		},
		{
			name:     "palette with nested blocks",
			input:    `palette{base="#191724"surface="#1f1d2e"highlight{low="#21202e"}}`,
			expected: `palette { base = "#191724" surface = "#1f1d2e" highlight { low = "#21202e" } }`,
		},
		{
			name: "already formatted stays same",
			input: `meta {
  name = "Test"
}
`,
			expected: `meta {
  name = "Test"
}
`,
		},
		{
			name:     "extra whitespace normalized",
			input:    `meta   {   name   =   "Test"   }`,
			expected: `meta { name = "Test" }`,
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name: "multiple blocks",
			input: `meta{name="Test"}
palette{base="#191724"}
theme{background=palette.base}`,
			expected: `meta { name = "Test" }
palette { base = "#191724" }
theme { background = palette.base }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := format(tt.input)
			if err != nil {
				t.Fatalf("format() error = %v", err)
			}

			// Normalize line endings for comparison
			result = strings.TrimSuffix(result, "\n")
			expected := strings.TrimSuffix(tt.expected, "\n")

			if result != expected {
				t.Errorf("format() = %q, want %q", result, expected)
			}
		})
	}
}

func TestFormatInvalidHCL(t *testing.T) {
	// hclwrite.Format should handle partial/invalid HCL gracefully
	input := `meta { name = "Test"`
	_, err := format(input)
	// The function should not error even on incomplete HCL
	if err != nil {
		t.Errorf("format() on incomplete HCL should not error, got: %v", err)
	}
}
