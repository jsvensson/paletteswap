package format

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
		{
			name: "multiple blank lines collapsed to one",
			input: "meta { name = \"Test\" }\n\n\n\npalette { base = \"#191724\" }",
			expected: "meta { name = \"Test\" }\n\npalette { base = \"#191724\" }",
		},
		{
			name: "many blank lines collapsed to one",
			input: "meta { name = \"Test\" }\n\n\n\n\n\n\npalette { base = \"#191724\" }",
			expected: "meta { name = \"Test\" }\n\npalette { base = \"#191724\" }",
		},
		{
			name: "single blank line preserved",
			input: "meta { name = \"Test\" }\n\npalette { base = \"#191724\" }",
			expected: "meta { name = \"Test\" }\n\npalette { base = \"#191724\" }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Format(tt.input)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Normalize line endings for comparison
			result = strings.TrimSuffix(result, "\n")
			expected := strings.TrimSuffix(tt.expected, "\n")

			if result != expected {
				t.Errorf("Format() = %q, want %q", result, expected)
			}
		})
	}
}

func TestFormatInvalidHCL(t *testing.T) {
	// hclwrite.Format should handle partial/invalid HCL gracefully
	input := `meta { name = "Test"`
	_, err := Format(input)
	// The function should not error even on incomplete HCL
	if err != nil {
		t.Errorf("Format() on incomplete HCL should not error, got: %v", err)
	}
}
