package lsp

import (
	"sort"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// themeForCompletion is a valid theme file used to produce an AnalysisResult
// for completion tests.
const themeForCompletion = `
meta {
  name       = "Test Theme"
  author     = "Test Author"
  appearance = "dark"
}

palette {
  base    = "#191724"
  surface = "#1f1d2e"
  love    = "#eb6f92"
  gold    = "#f6c177"

  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

theme {
  background = palette.base
  foreground = palette.surface
  cursor     = palette.love
}

ansi {
  black   = palette.base
  red     = palette.love
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}

syntax {
  keyword = palette.love
  string  = palette.gold
  comment {
    color  = palette.surface
    italic = true
  }
}
`

func completionLabels(items []protocol.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	sort.Strings(labels)
	return labels
}

func hasLabel(items []protocol.CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func TestCompletion_PaletteTopLevel(t *testing.T) {
	result := Analyze("test.pstheme", themeForCompletion)
	if result.Palette == nil {
		t.Fatal("expected non-nil palette from analysis")
	}

	// Cursor after "palette." on a theme attribute value line.
	// Line: "  cursor     = palette."
	// In themeForCompletion, the theme block starts around line 33 (0-indexed: 32).
	// We place cursor at a synthetic position inside the theme block.
	content := themeForCompletion

	// Find "palette." in the theme block — let's place the cursor at a value position
	// where we type "palette." and want completions.
	// We'll construct a modified content where the cursor is after "palette."
	modifiedContent := `
meta {
  name       = "Test Theme"
  author     = "Test Author"
  appearance = "dark"
}

palette {
  base    = "#191724"
  surface = "#1f1d2e"
  love    = "#eb6f92"
  gold    = "#f6c177"

  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

theme {
  background = palette.base
  foreground = palette.surface
  cursor     = palette.
}
`
	// "cursor     = palette." is on a line in the theme block.
	// Count lines to find it. The line with "palette." at the end.
	_ = content // use original for analysis result
	lines := splitLines(modifiedContent)
	var targetLine uint32
	for i, line := range lines {
		if len(line) > 0 && line[len(line)-1] == '.' {
			// "  cursor     = palette."
			if len(line) >= 8 && line[len(line)-8:] == "palette." {
				targetLine = uint32(i)
				break
			}
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: uint32(len(lines[targetLine])),
	}

	items := complete(result, modifiedContent, pos)

	if len(items) == 0 {
		t.Fatal("expected completion items for palette., got none")
	}

	// Should include top-level palette children: base, surface, love, gold, highlight
	expectedLabels := []string{"base", "surface", "love", "gold", "highlight"}
	for _, label := range expectedLabels {
		if !hasLabel(items, label) {
			t.Errorf("expected completion item %q, not found in results", label)
		}
	}

	// Verify kind is Color for items with resolved colors
	for _, item := range items {
		if item.Label != "highlight" {
			if item.Kind == nil || *item.Kind != protocol.CompletionItemKindColor {
				t.Errorf("expected CompletionItemKindColor for %q", item.Label)
			}
			if item.Detail == nil || *item.Detail == "" {
				t.Errorf("expected non-empty Detail (hex) for %q", item.Label)
			}
		}
	}
}

func TestCompletion_PaletteNested(t *testing.T) {
	// Analyze valid content to get the palette tree with highlight.{low, high}.
	validContent := `
palette {
  base    = "#191724"
  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

theme {
  background = palette.highlight.low
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", validContent)
	if result.Palette == nil {
		t.Fatal("expected non-nil palette from analysis")
	}

	// Simulate editing: the user has typed "palette.highlight." and wants completions.
	// We use content where the cursor is right after the trailing dot.
	editingContent := `
palette {
  base    = "#191724"
  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

theme {
  background = palette.highlight.
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	lines := splitLines(editingContent)
	var targetLine uint32
	found := false
	for i, line := range lines {
		if strings.Contains(line, "palette.highlight.") {
			targetLine = uint32(i)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("could not find 'palette.highlight.' in test content")
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: uint32(len(lines[targetLine])),
	}

	items := complete(result, editingContent, pos)

	if len(items) == 0 {
		t.Fatal("expected completion items for palette.highlight., got none")
	}

	// Should include highlight children: low, high
	if !hasLabel(items, "low") {
		t.Error("expected completion item 'low'")
	}
	if !hasLabel(items, "high") {
		t.Error("expected completion item 'high'")
	}

	// "color" is a reserved keyword — it must NOT be suggested as a completion
	if hasLabel(items, "color") {
		t.Error("should not suggest reserved keyword 'color' as palette completion")
	}

	// Exactly two items expected: low, high
	if len(items) != 2 {
		t.Errorf("expected exactly 2 completion items, got %d: %v", len(items), completionLabels(items))
	}
}

func TestCompletion_ANSIMissingNames(t *testing.T) {
	// Partial ansi block missing some colors
	content := `
palette {
  base = "#191724"
}

ansi {
  black = palette.base
  red   = "#ff0000"

}
`
	result := Analyze("test.pstheme", content)

	// Cursor on the blank line inside ansi block (line with just spaces or empty)
	lines := splitLines(content)
	// Find the empty line inside the ansi block (right before the closing brace)
	var targetLine uint32
	inAnsi := false
	for i, line := range lines {
		if len(line) >= 4 && line[:4] == "ansi" {
			inAnsi = true
			continue
		}
		if inAnsi && trimSpace(line) == "" {
			targetLine = uint32(i)
			break
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: 2, // indented position, as if typing a new attribute name
	}

	items := complete(result, content, pos)

	if len(items) == 0 {
		t.Fatal("expected ANSI completion items, got none")
	}

	// Should NOT include "black" and "red" (already defined)
	if hasLabel(items, "black") {
		t.Error("should not suggest already-defined 'black'")
	}
	if hasLabel(items, "red") {
		t.Error("should not suggest already-defined 'red'")
	}

	// Should include some missing ANSI colors
	if !hasLabel(items, "green") {
		t.Error("expected 'green' in ANSI completions")
	}
	if !hasLabel(items, "bright_white") {
		t.Error("expected 'bright_white' in ANSI completions")
	}

	// Verify kind is Constant
	for _, item := range items {
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindConstant {
			t.Errorf("expected CompletionItemKindConstant for ANSI item %q", item.Label)
		}
	}
}

func TestCompletion_TopLevelBlocks(t *testing.T) {
	content := `
palette {
  base = "#191724"
}

`
	result := Analyze("test.pstheme", content)

	// Cursor on the last blank line, at root level
	lines := splitLines(content)
	targetLine := uint32(len(lines) - 1)

	pos := protocol.Position{
		Line:      targetLine,
		Character: 0,
	}

	items := complete(result, content, pos)

	if len(items) == 0 {
		t.Fatal("expected top-level block completion items, got none")
	}

	expectedBlocks := []string{"meta", "palette", "theme", "syntax", "ansi"}
	for _, block := range expectedBlocks {
		if !hasLabel(items, block) {
			t.Errorf("expected top-level block completion %q", block)
		}
	}
}

func TestCompletion_StyleAttributes(t *testing.T) {
	content := `
palette {
  base = "#191724"
  love = "#eb6f92"
}

syntax {
  comment {
    color = palette.base

  }
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", content)

	// Find the blank line inside the comment block
	lines := splitLines(content)
	var targetLine uint32
	inComment := false
	for i, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "comment {" {
			inComment = true
			continue
		}
		if inComment && trimmed == "" {
			targetLine = uint32(i)
			break
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: 4, // indented inside the style block
	}

	items := complete(result, content, pos)

	if len(items) == 0 {
		t.Fatal("expected style attribute completions, got none")
	}

	// "color" is already defined, should NOT appear
	if hasLabel(items, "color") {
		t.Error("should not suggest already-defined 'color'")
	}

	// Should include bold, italic, underline
	if !hasLabel(items, "bold") {
		t.Error("expected 'bold' in style completions")
	}
	if !hasLabel(items, "italic") {
		t.Error("expected 'italic' in style completions")
	}
	if !hasLabel(items, "underline") {
		t.Error("expected 'underline' in style completions")
	}

	// Verify kind is Keyword
	for _, item := range items {
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindKeyword {
			t.Errorf("expected CompletionItemKindKeyword for style item %q", item.Label)
		}
	}
}

func TestCompletion_Functions(t *testing.T) {
	content := `
palette {
  base = "#191724"
}

theme {
  background =
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", content)

	// Find the line with "background = " and put cursor after the equals sign
	lines := splitLines(content)
	var targetLine uint32
	var targetChar uint32
	for i, line := range lines {
		trimmed := trimSpace(line)
		if len(trimmed) >= 13 && trimmed[:13] == "background = " {
			targetLine = uint32(i)
			targetChar = uint32(len(line))
			break
		}
		// Also check for "background =" without trailing space
		if trimmed == "background =" {
			targetLine = uint32(i)
			targetChar = uint32(len(line))
			break
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: targetChar,
	}

	items := complete(result, content, pos)

	// Should include function completions
	if !hasLabel(items, "brighten") {
		t.Error("expected 'brighten' function completion")
	}
	if !hasLabel(items, "darken") {
		t.Error("expected 'darken' function completion")
	}

	// Functions should also include palette. as a trigger
	if !hasLabel(items, "palette") {
		t.Error("expected 'palette' value completion for palette references")
	}
}

// trimSpace trims leading and trailing whitespace from a string.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func TestCompletion_PaletteWithSyntaxError(t *testing.T) {
	// This tests that palette completion works even when there are syntax errors
	// elsewhere in the file (e.g., incomplete palette. reference in syntax block)
	content := `
palette {
  base    = "#191724"
  surface = "#1f1d2e"
  love    = "#eb6f92"

  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

syntax {
  keyword = palette.
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", content)

	// Palette should be built even with syntax errors
	if result.Palette == nil {
		t.Fatal("expected palette tree to be built despite syntax errors")
	}

	// Find the line with "palette." in the syntax block
	lines := splitLines(content)
	var targetLine uint32
	for i, line := range lines {
		if strings.Contains(line, "keyword = palette.") {
			targetLine = uint32(i)
			break
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: uint32(len(lines[targetLine])),
	}

	items := complete(result, content, pos)

	if len(items) == 0 {
		t.Fatal("expected completion items for palette. even with syntax errors, got none")
	}

	// Should include top-level palette children
	expectedLabels := []string{"base", "surface", "love", "highlight"}
	for _, label := range expectedLabels {
		if !hasLabel(items, label) {
			t.Errorf("expected completion item %q, not found in results", label)
		}
	}
}

func TestCompletion_PaletteNestedWithSyntaxError(t *testing.T) {
	// Test nested palette completion works even with incomplete references elsewhere
	content := `
palette {
  base    = "#191724"

  highlight {
    color = "#524f67"
    low   = "#21202e"
    high  = "#6e6a86"
  }
}

theme {
  background = palette.
}

syntax {
  comment = palette.highlight.
}

ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", content)

	if result.Palette == nil {
		t.Fatal("expected palette tree to be built despite syntax errors")
	}

	// Test nested completion: palette.highlight.
	lines := splitLines(content)
	var targetLine uint32
	for i, line := range lines {
		if strings.Contains(line, "comment = palette.highlight.") {
			targetLine = uint32(i)
			break
		}
	}

	pos := protocol.Position{
		Line:      targetLine,
		Character: uint32(len(lines[targetLine])),
	}

	items := complete(result, content, pos)

	if len(items) == 0 {
		t.Fatal("expected completion items for palette.highlight., got none")
	}

	// Should include highlight children: low, high
	if !hasLabel(items, "low") {
		t.Error("expected completion item 'low' for palette.highlight.")
	}
	if !hasLabel(items, "high") {
		t.Error("expected completion item 'high' for palette.highlight.")
	}

	// "color" is a reserved keyword — it must NOT be suggested
	if hasLabel(items, "color") {
		t.Error("should not suggest reserved keyword 'color' as palette completion")
	}
}
