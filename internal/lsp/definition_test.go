package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDefinition_PaletteBase(t *testing.T) {
	// Document with palette.base reference in theme block
	content := `palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`
	result := Analyze("test.pstheme", content)

	// Verify the symbol table has palette.base
	symRange, ok := result.Symbols["palette.base"]
	if !ok {
		t.Fatal("expected palette.base in symbol table")
	}

	// Position cursor on "palette.base" in the theme block (line 5, somewhere in "palette.base")
	// Line 5 is "  background = palette.base"
	// "palette.base" starts at character 15
	pos := protocol.Position{Line: 5, Character: 17} // inside "palette.base"
	uri := "file:///test.pstheme"

	loc := definition(result, content, uri, pos)
	if loc == nil {
		t.Fatal("expected non-nil definition location for palette.base reference")
	}

	if loc.URI != protocol.DocumentUri(uri) {
		t.Errorf("URI = %q, want %q", loc.URI, uri)
	}

	if loc.Range != symRange {
		t.Errorf("Range = %v, want %v", loc.Range, symRange)
	}
}

func TestDefinition_NestedPalette(t *testing.T) {
	// Document with nested palette reference
	content := `palette {
  highlight {
    low  = "#21202e"
    high = "#524f67"
  }
}

theme {
  background = palette.highlight.low
}
`
	result := Analyze("test.pstheme", content)

	symRange, ok := result.Symbols["palette.highlight.low"]
	if !ok {
		t.Fatal("expected palette.highlight.low in symbol table")
	}

	// Line 8 is "  background = palette.highlight.low"
	// "palette.highlight.low" starts at character 15
	pos := protocol.Position{Line: 8, Character: 20} // inside "palette.highlight.low"
	uri := "file:///test.pstheme"

	loc := definition(result, content, uri, pos)
	if loc == nil {
		t.Fatal("expected non-nil definition location for palette.highlight.low reference")
	}

	if loc.URI != protocol.DocumentUri(uri) {
		t.Errorf("URI = %q, want %q", loc.URI, uri)
	}

	if loc.Range != symRange {
		t.Errorf("Range = %v, want %v", loc.Range, symRange)
	}
}

func TestDefinition_HexLiteral(t *testing.T) {
	// Cursor on a hex literal should return nil
	content := `palette {
  base = "#191724"
}

theme {
  cursor = "#ff0000"
}
`
	result := Analyze("test.pstheme", content)

	// Line 5 is '  cursor = "#ff0000"'
	// Position on the hex literal
	pos := protocol.Position{Line: 5, Character: 14} // inside "#ff0000"
	uri := "file:///test.pstheme"

	loc := definition(result, content, uri, pos)
	if loc != nil {
		t.Errorf("expected nil for hex literal, got %+v", loc)
	}
}

func TestDefinition_PlainText(t *testing.T) {
	// Cursor on plain text (not a palette reference) should return nil
	content := `palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`
	result := Analyze("test.pstheme", content)

	// Line 0 is "palette {"
	pos := protocol.Position{Line: 0, Character: 2} // on "palette" keyword in block header
	uri := "file:///test.pstheme"

	loc := definition(result, content, uri, pos)
	if loc != nil {
		t.Errorf("expected nil for plain text, got %+v", loc)
	}
}

func TestDefinition_NilResult(t *testing.T) {
	uri := "file:///test.pstheme"
	pos := protocol.Position{Line: 0, Character: 0}

	loc := definition(nil, "", uri, pos)
	if loc != nil {
		t.Errorf("expected nil for nil result, got %+v", loc)
	}
}
