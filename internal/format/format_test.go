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
		{
			name: "blank line after opening brace removed",
			input: "palette {\n\n  base = \"#191724\"\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "blank line before closing brace removed",
			input: "palette {\n  base = \"#191724\"\n\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "blank lines after and before braces both removed",
			input: "palette {\n\n  base = \"#191724\"\n\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "nested block blank lines removed",
			input: "palette {\n\n  highlight {\n\n    low = \"#21202e\"\n\n  }\n\n}",
			expected: "palette {\n  highlight {\n    low = \"#21202e\"\n  }\n}",
		},
		{
			name: "ansi block already in order stays same",
			input: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
			expected: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
		},
		{
			name: "ansi block misordered gets reordered",
			input: `ansi {
  white          = palette.text
  bright_cyan    = palette.foam
  red            = palette.love
  bright_black   = palette.muted
  green          = palette.pine
  magenta        = palette.iris
  bright_white   = palette.text
  yellow         = palette.gold
  blue           = palette.foam
  bright_red     = palette.love
  black          = palette.overlay
  cyan           = palette.foam
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
}
`,
			expected: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
		},
		{
			name: "ansi block comments travel with their attribute",
			input: `ansi {
  white          = palette.text
  red            = palette.love
  # bright colors
  bright_black   = palette.muted
  black          = palette.overlay
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
			expected: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  # bright colors
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
		},
		{
			name: "ansi block inline comments preserved",
			input: `ansi {
  white          = palette.text # foreground
  red            = palette.love
  black          = palette.overlay # background
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
			expected: `ansi {
  black          = palette.overlay # background
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text # foreground
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
		},
		{
			name: "no ansi block unchanged",
			input: `meta {
  name = "Test"
}

palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`,
			expected: `meta {
  name = "Test"
}

palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`,
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
