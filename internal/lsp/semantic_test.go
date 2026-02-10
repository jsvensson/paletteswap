package lsp

import (
	"reflect"
	"testing"
)

func TestEncodeTokens_Empty(t *testing.T) {
	result := encodeTokens([]SemanticToken{})
	expected := []uint32{}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("encodeTokens([]) = %v, want %v", result, expected)
	}
}

func TestEncodeTokens_SingleToken(t *testing.T) {
	tokens := []SemanticToken{
		{Line: 2, StartChar: 5, Length: 7, Type: 0, Modifiers: 0},
	}
	result := encodeTokens(tokens)
	expected := []uint32{2, 5, 7, 0, 0}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("encodeTokens() = %v, want %v", result, expected)
	}
}

func TestEncodeTokens_MultipleTokensSameLine(t *testing.T) {
	tokens := []SemanticToken{
		{Line: 0, StartChar: 0, Length: 7, Type: 0, Modifiers: 0}, // "palette"
		{Line: 0, StartChar: 8, Length: 4, Type: 1, Modifiers: 1}, // "base"
	}
	result := encodeTokens(tokens)
	// Second token: deltaLine=0, deltaStart=8-0=8
	expected := []uint32{0, 0, 7, 0, 0, 0, 8, 4, 1, 1}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("encodeTokens() = %v, want %v", result, expected)
	}
}

func TestEncodeTokens_MultipleTokensDifferentLines(t *testing.T) {
	tokens := []SemanticToken{
		{Line: 0, StartChar: 0, Length: 7, Type: 0, Modifiers: 0}, // line 0
		{Line: 2, StartChar: 2, Length: 4, Type: 1, Modifiers: 0}, // line 2
	}
	result := encodeTokens(tokens)
	// Second token: deltaLine=2-0=2, deltaStart=2 (new line, not relative)
	expected := []uint32{0, 0, 7, 0, 0, 2, 2, 4, 1, 0}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("encodeTokens() = %v, want %v", result, expected)
	}
}

func TestEncodeTokens_SortsTokens(t *testing.T) {
	// Tokens in wrong order
	tokens := []SemanticToken{
		{Line: 1, StartChar: 0, Length: 4, Type: 1, Modifiers: 0},
		{Line: 0, StartChar: 0, Length: 7, Type: 0, Modifiers: 0},
	}
	result := encodeTokens(tokens)
	// Should be sorted: line 0 first, then line 1
	expected := []uint32{0, 0, 7, 0, 0, 1, 0, 4, 1, 0}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("encodeTokens() = %v, want %v", result, expected)
	}
}

func TestSemanticTokensFull_Empty(t *testing.T) {
	content := ``
	result := semanticTokensFull(content)
	if len(result) != 0 {
		t.Errorf("semanticTokensFull(\"\") = %v, want empty", result)
	}
}

func TestSemanticTokensFull_SimplePalette(t *testing.T) {
	content := `palette {
  base = "#191724"
}`
	result := semanticTokensFull(content)

	// Should have: "palette" (keyword), "base" (property)
	// That's 2 tokens = 10 integers
	// Note: String literals are not currently tokenized
	if len(result) != 10 {
		t.Errorf("semanticTokensFull() returned %d integers, want 10", len(result))
	}
}

func TestSemanticTokensFull_WithPaletteReference(t *testing.T) {
	content := `palette {
  base = "#191724"
}
theme {
  background = palette.base
}`
	result := semanticTokensFull(content)

	// Should have: palette(keyword), base(property),
	//              theme(keyword), background(property), palette(namespace), base(property)
	// That's 6 tokens = 30 integers
	if len(result) != 30 {
		t.Errorf("semanticTokensFull() returned %d integers, want 30", len(result))
	}
}

func TestSemanticTokensFull_WithFunction(t *testing.T) {
	content := `palette {
  base = "#191724"
  surface = brighten(base, 0.1)
}`
	result := semanticTokensFull(content)

	// Should have: palette(keyword), base(property),
	//              surface(property), brighten(function), 0.1(number)
	// That's 5 tokens = 25 integers
	// Note: String literals in templates and non-palette references are not currently tokenized
	if len(result) != 25 {
		t.Errorf("semanticTokensFull() returned %d integers, want 25", len(result))
	}
}

func TestSemanticTokensFull_ParseError(t *testing.T) {
	content := `palette {`
	result := semanticTokensFull(content)
	if len(result) != 0 {
		t.Errorf("semanticTokensFull(parse error) = %v, want empty", result)
	}
}

func TestSemanticTokensFull_CompleteTheme(t *testing.T) {
	content := `meta {
  name = "Test Theme"
}

palette {
  base    = "#191724"
  surface = "#1f1d2e"

  highlight {
    low  = "#21202e"
    high = "#524f67"
  }
}

theme {
  background = palette.base
  foreground = palette.surface
}

syntax {
  keyword = palette.highlight.low
  comment {
    color  = palette.highlight.high
    italic = true
  }
}`

	result := semanticTokensFull(content)

	// Verify we got some tokens back
	if len(result) == 0 {
		t.Fatal("semanticTokensFull() returned empty for valid theme")
	}

	// Verify the data length is a multiple of 5
	if len(result)%5 != 0 {
		t.Errorf("semantic tokens data length %d is not a multiple of 5", len(result))
	}

	// Should have at least: meta, name, palette, base, surface, highlight, low, high,
	// theme, background, foreground, syntax, keyword, comment, color, italic
	// That's at least 16 tokens = 80 integers
	if len(result) < 80 {
		t.Errorf("semanticTokensFull() returned %d integers, expected at least 80", len(result))
	}
}
