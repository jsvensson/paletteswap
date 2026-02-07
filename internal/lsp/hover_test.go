package lsp

import (
	"strings"
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestHover_PaletteReference(t *testing.T) {
	// Simulate a document where "palette.base" is at a known position
	content := `palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`
	result := Analyze("test.pstheme", content)

	// Find the ColorLocation that is a reference (palette.base in theme block)
	var refLoc *ColorLocation
	for i, cl := range result.Colors {
		if cl.IsRef {
			refLoc = &result.Colors[i]
			break
		}
	}
	if refLoc == nil {
		t.Fatal("expected to find a palette reference ColorLocation")
	}

	// Hover at a position within the reference range
	pos := protocol.Position{
		Line:      refLoc.Range.Start.Line,
		Character: refLoc.Range.Start.Character + 2, // somewhere inside "palette.base"
	}

	h := hover(result, content, pos)
	if h == nil {
		t.Fatal("expected non-nil hover result for palette reference")
	}

	mc, ok := h.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", h.Contents)
	}

	if mc.Kind != protocol.MarkupKindMarkdown {
		t.Errorf("expected markdown kind, got %q", mc.Kind)
	}

	// Should contain the source text (palette.base)
	if !strings.Contains(mc.Value, "palette.base") {
		t.Errorf("hover content should contain source text 'palette.base', got:\n%s", mc.Value)
	}

	// Should contain the hex value
	if !strings.Contains(mc.Value, "#191724") {
		t.Errorf("hover content should contain hex '#191724', got:\n%s", mc.Value)
	}

	// Should contain the RGB value
	if !strings.Contains(mc.Value, "rgb(25, 23, 36)") {
		t.Errorf("hover content should contain 'rgb(25, 23, 36)', got:\n%s", mc.Value)
	}
}

func TestHover_HexLiteral(t *testing.T) {
	content := `palette {
  love = "#eb6f92"
}
`
	result := Analyze("test.pstheme", content)

	// Find the hex literal ColorLocation
	var hexLoc *ColorLocation
	for i, cl := range result.Colors {
		if !cl.IsRef {
			hexLoc = &result.Colors[i]
			break
		}
	}
	if hexLoc == nil {
		t.Fatal("expected to find a hex literal ColorLocation")
	}

	pos := protocol.Position{
		Line:      hexLoc.Range.Start.Line,
		Character: hexLoc.Range.Start.Character + 1, // inside the hex literal
	}

	h := hover(result, content, pos)
	if h == nil {
		t.Fatal("expected non-nil hover result for hex literal")
	}

	mc, ok := h.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", h.Contents)
	}

	// Should NOT have a bold header (no reference name for hex literals)
	if strings.Contains(mc.Value, "**") {
		t.Errorf("hex literal hover should not have bold header, got:\n%s", mc.Value)
	}

	// Should contain hex
	if !strings.Contains(mc.Value, "#eb6f92") {
		t.Errorf("hover content should contain '#eb6f92', got:\n%s", mc.Value)
	}

	// Should contain RGB
	if !strings.Contains(mc.Value, "rgb(235, 111, 146)") {
		t.Errorf("hover content should contain 'rgb(235, 111, 146)', got:\n%s", mc.Value)
	}
}

func TestHover_NoColor(t *testing.T) {
	content := `palette {
  base = "#191724"
}
`
	result := Analyze("test.pstheme", content)

	// Position on "palette {" keyword, which is not a color location
	pos := protocol.Position{
		Line:      0,
		Character: 0,
	}

	h := hover(result, content, pos)
	if h != nil {
		t.Errorf("expected nil hover for non-color position, got: %+v", h)
	}
}

func TestPosInRange(t *testing.T) {
	r := protocol.Range{
		Start: protocol.Position{Line: 5, Character: 10},
		End:   protocol.Position{Line: 5, Character: 22},
	}

	tests := []struct {
		name string
		pos  protocol.Position
		want bool
	}{
		{"before range", protocol.Position{Line: 5, Character: 9}, false},
		{"at start", protocol.Position{Line: 5, Character: 10}, true},
		{"in middle", protocol.Position{Line: 5, Character: 15}, true},
		{"at end (exclusive)", protocol.Position{Line: 5, Character: 22}, false},
		{"after range", protocol.Position{Line: 5, Character: 23}, false},
		{"line before", protocol.Position{Line: 4, Character: 15}, false},
		{"line after", protocol.Position{Line: 6, Character: 15}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := posInRange(tt.pos, r)
			if got != tt.want {
				t.Errorf("posInRange(%v, %v) = %v, want %v", tt.pos, r, got, tt.want)
			}
		})
	}
}

func TestHover_FunctionDirect(t *testing.T) {
	// Test hover function directly with crafted AnalysisResult
	c, _ := color.ParseHex("#ff0000")
	result := &AnalysisResult{
		Colors: []ColorLocation{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 5},
					End:   protocol.Position{Line: 2, Character: 17},
				},
				Color: c,
				IsRef: true,
			},
		},
	}

	content := "line 0\nline 1\n     palette.red is here\nline 3\n"

	// Position inside the color range
	pos := protocol.Position{Line: 2, Character: 10}
	h := hover(result, content, pos)
	if h == nil {
		t.Fatal("expected hover result")
	}

	mc := h.Contents.(protocol.MarkupContent)
	// The source text from the range should be "palette.red"
	if !strings.Contains(mc.Value, "palette.red") {
		t.Errorf("expected source text 'palette.red' in hover, got:\n%s", mc.Value)
	}
	if !strings.Contains(mc.Value, "#ff0000") {
		t.Errorf("expected '#ff0000' in hover, got:\n%s", mc.Value)
	}
	if !strings.Contains(mc.Value, "rgb(255, 0, 0)") {
		t.Errorf("expected 'rgb(255, 0, 0)' in hover, got:\n%s", mc.Value)
	}

	// Position outside the color range
	pos = protocol.Position{Line: 0, Character: 0}
	h = hover(result, content, pos)
	if h != nil {
		t.Error("expected nil hover for position outside color range")
	}
}
