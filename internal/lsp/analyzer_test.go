package lsp

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

const validTheme = `
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
  pine    = "#31748f"
  foam    = "#9ccfd8"
}

theme {
  background = palette.base
  foreground = palette.foam
  cursor     = palette.love
}

syntax {
  keyword = palette.pine
  string  = palette.gold
  comment {
    color  = palette.surface
    italic = true
  }
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
`

func TestAnalyze_ValidTheme(t *testing.T) {
	result := Analyze("test.pstheme", validTheme)

	if len(result.Diagnostics) != 0 {
		for _, d := range result.Diagnostics {
			t.Logf("  diagnostic: [%v] %s", *d.Severity, d.Message)
		}
		t.Fatalf("expected 0 diagnostics, got %d", len(result.Diagnostics))
	}

	if result.Palette == nil {
		t.Fatal("expected non-nil palette")
	}

	// Verify palette has expected entries
	base, err := result.Palette.Lookup([]string{"base"})
	if err != nil {
		t.Fatalf("Lookup(base) error: %v", err)
	}
	if base.Hex() != "#191724" {
		t.Errorf("palette.base = %q, want %q", base.Hex(), "#191724")
	}
}

func TestAnalyze_SyntaxError(t *testing.T) {
	content := `
palette {
  base = "#191724"
  this is not valid HCL!!!!
}
`
	result := Analyze("test.pstheme", content)

	if len(result.Diagnostics) == 0 {
		t.Fatal("expected at least 1 diagnostic for syntax error")
	}

	// All syntax errors should be error-level
	for _, d := range result.Diagnostics {
		if d.Severity == nil || *d.Severity != protocol.DiagnosticSeverityError {
			t.Errorf("expected error severity, got %v", d.Severity)
		}
	}
}

func TestAnalyze_UndefinedPaletteRef(t *testing.T) {
	content := `
palette {
  base = "#191724"
}

theme {
  background = palette.nonexistent
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

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			if strings.Contains(d.Message, "nonexistent") || strings.Contains(d.Message, "background") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected error diagnostic mentioning undefined palette reference")
		for _, d := range result.Diagnostics {
			t.Logf("  diagnostic: [%v] %s", *d.Severity, d.Message)
		}
	}
}

func TestAnalyze_MissingANSI(t *testing.T) {
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

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityWarning {
			if strings.Contains(d.Message, "missing") || strings.Contains(d.Message, "Missing") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected warning diagnostic for missing ANSI colors")
		for _, d := range result.Diagnostics {
			t.Logf("  diagnostic: [%v] %s", *d.Severity, d.Message)
		}
	}
}

func TestAnalyze_MissingPalette(t *testing.T) {
	content := `
meta {
  name = "test"
}

theme {
  background = "#000000"
}
`
	result := Analyze("test.pstheme", content)

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			if strings.Contains(d.Message, "palette") || strings.Contains(d.Message, "Palette") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected error diagnostic for missing palette block")
		for _, d := range result.Diagnostics {
			t.Logf("  diagnostic: [%v] %s", *d.Severity, d.Message)
		}
	}
}

func TestAnalyze_InvalidHex(t *testing.T) {
	content := `
palette {
  bad = "not-a-color"
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

	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			if strings.Contains(d.Message, "bad") || strings.Contains(d.Message, "hex") || strings.Contains(d.Message, "invalid") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected error diagnostic for invalid hex color")
		for _, d := range result.Diagnostics {
			t.Logf("  diagnostic: [%v] %s", *d.Severity, d.Message)
		}
	}
}

func TestAnalyze_SymbolTable(t *testing.T) {
	content := `
palette {
  base = "#191724"
  love = "#eb6f92"

  highlight {
    low  = "#21202e"
    high = "#524f67"
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

	expectedSymbols := []string{
		"palette.base",
		"palette.love",
		"palette.highlight.low",
		"palette.highlight.high",
	}

	for _, sym := range expectedSymbols {
		if _, ok := result.Symbols[sym]; !ok {
			t.Errorf("expected symbol %q in symbol table, but it was not found", sym)
		}
	}

	// Check that the range is reasonable (line > 0 for all of these since they're not at the start)
	for sym, rng := range result.Symbols {
		t.Logf("symbol %q: line %d, col %d", sym, rng.Start.Line, rng.Start.Character)
		// All palette entries are past line 0
		if rng.Start.Line == 0 && rng.Start.Character == 0 && rng.End.Line == 0 && rng.End.Character == 0 {
			t.Errorf("symbol %q has zero range, expected real position", sym)
		}
	}
}

func TestAnalyze_ColorLocations(t *testing.T) {
	content := `
palette {
  base = "#191724"
  love = "#eb6f92"
}

theme {
  background = palette.base
  cursor     = "#ff0000"
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

	if len(result.Colors) == 0 {
		t.Fatal("expected at least one color location")
	}

	// Check that we have both ref and non-ref colors
	hasRef := false
	hasLiteral := false
	for _, cl := range result.Colors {
		if cl.IsRef {
			hasRef = true
		} else {
			hasLiteral = false
			// Actually even hex literals in palette are not refs
			hasLiteral = true
		}
		t.Logf("color %s at line %d (ref=%v)", cl.Color.Hex(), cl.Range.Start.Line, cl.IsRef)
	}

	if !hasRef {
		t.Error("expected at least one palette reference color location (IsRef=true)")
	}
	if !hasLiteral {
		t.Error("expected at least one literal color location (IsRef=false)")
	}
}
