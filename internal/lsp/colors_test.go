package lsp

import (
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestColorToLSP(t *testing.T) {
	tests := []struct {
		name  string
		input color.Color
		want  protocol.Color
	}{
		{
			name:  "pure red",
			input: color.Color{R: 255, G: 0, B: 0},
			want:  protocol.Color{Red: 1.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
		},
		{
			name:  "pure green",
			input: color.Color{R: 0, G: 255, B: 0},
			want:  protocol.Color{Red: 0.0, Green: 1.0, Blue: 0.0, Alpha: 1.0},
		},
		{
			name:  "pure blue",
			input: color.Color{R: 0, G: 0, B: 255},
			want:  protocol.Color{Red: 0.0, Green: 0.0, Blue: 1.0, Alpha: 1.0},
		},
		{
			name:  "black",
			input: color.Color{R: 0, G: 0, B: 0},
			want:  protocol.Color{Red: 0.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
		},
		{
			name:  "white",
			input: color.Color{R: 255, G: 255, B: 255},
			want:  protocol.Color{Red: 1.0, Green: 1.0, Blue: 1.0, Alpha: 1.0},
		},
		{
			name:  "mid gray",
			input: color.Color{R: 128, G: 128, B: 128},
			want:  protocol.Color{Red: float32(128) / 255.0, Green: float32(128) / 255.0, Blue: float32(128) / 255.0, Alpha: 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorToLSP(tt.input)
			if got.Red != tt.want.Red {
				t.Errorf("Red: got %f, want %f", got.Red, tt.want.Red)
			}
			if got.Green != tt.want.Green {
				t.Errorf("Green: got %f, want %f", got.Green, tt.want.Green)
			}
			if got.Blue != tt.want.Blue {
				t.Errorf("Blue: got %f, want %f", got.Blue, tt.want.Blue)
			}
			if got.Alpha != tt.want.Alpha {
				t.Errorf("Alpha: got %f, want %f", got.Alpha, tt.want.Alpha)
			}
		})
	}
}

func TestDocumentColors(t *testing.T) {
	red, _ := color.ParseHex("#ff0000")
	green, _ := color.ParseHex("#00ff00")
	blue, _ := color.ParseHex("#0000ff")

	result := &AnalysisResult{
		Colors: []ColorLocation{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 10},
					End:   protocol.Position{Line: 1, Character: 20},
				},
				Color: red,
				IsRef: false,
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 10},
					End:   protocol.Position{Line: 2, Character: 22},
				},
				Color: green,
				IsRef: true,
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 10},
					End:   protocol.Position{Line: 3, Character: 20},
				},
				Color: blue,
				IsRef: false,
			},
		},
	}

	infos := documentColors(result)

	if len(infos) != 3 {
		t.Fatalf("expected 3 ColorInformation items, got %d", len(infos))
	}

	// Check first item (red)
	if infos[0].Color.Red != 1.0 || infos[0].Color.Green != 0.0 || infos[0].Color.Blue != 0.0 {
		t.Errorf("item 0: expected red, got R=%f G=%f B=%f", infos[0].Color.Red, infos[0].Color.Green, infos[0].Color.Blue)
	}
	if infos[0].Color.Alpha != 1.0 {
		t.Errorf("item 0: expected alpha 1.0, got %f", infos[0].Color.Alpha)
	}
	if infos[0].Range.Start.Line != 1 || infos[0].Range.Start.Character != 10 {
		t.Errorf("item 0: unexpected range start")
	}

	// Check second item (green)
	if infos[1].Color.Red != 0.0 || infos[1].Color.Green != 1.0 || infos[1].Color.Blue != 0.0 {
		t.Errorf("item 1: expected green, got R=%f G=%f B=%f", infos[1].Color.Red, infos[1].Color.Green, infos[1].Color.Blue)
	}

	// Check third item (blue)
	if infos[2].Color.Red != 0.0 || infos[2].Color.Green != 0.0 || infos[2].Color.Blue != 1.0 {
		t.Errorf("item 2: expected blue, got R=%f G=%f B=%f", infos[2].Color.Red, infos[2].Color.Green, infos[2].Color.Blue)
	}
}

func TestDocumentColors_NilResult(t *testing.T) {
	infos := documentColors(nil)
	if infos == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 items, got %d", len(infos))
	}
}

func TestDocumentColors_EmptyColors(t *testing.T) {
	result := &AnalysisResult{
		Colors: []ColorLocation{},
	}
	infos := documentColors(result)
	if len(infos) != 0 {
		t.Errorf("expected 0 items, got %d", len(infos))
	}
}

func TestColorPresentation_HexLiteral(t *testing.T) {
	// Document content with a hex literal at the given range
	content := "palette {\n  base = \"#191724\"\n}\n"

	// The range covers the hex literal including quotes: "#191724"
	params := &protocol.ColorPresentationParams{
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 9},
			End:   protocol.Position{Line: 1, Character: 19},
		},
	}

	presentations := colorPresentation(content, params)

	if len(presentations) != 1 {
		t.Fatalf("expected 1 presentation for hex literal, got %d", len(presentations))
	}

	// The label should be the new hex value
	if presentations[0].Label != "#ff0000" {
		t.Errorf("expected label '#ff0000', got %q", presentations[0].Label)
	}

	// Should have a TextEdit to replace the old value
	if presentations[0].TextEdit == nil {
		t.Fatal("expected non-nil TextEdit for hex literal")
	}

	if presentations[0].TextEdit.NewText != "\"#ff0000\"" {
		t.Errorf("expected TextEdit.NewText '\"#ff0000\"', got %q", presentations[0].TextEdit.NewText)
	}

	// TextEdit range should match the params range
	if presentations[0].TextEdit.Range != params.Range {
		t.Errorf("expected TextEdit range to match params range")
	}
}

func TestColorPresentation_PaletteReference(t *testing.T) {
	// Document content with a palette reference at the given range
	content := "theme {\n  background = palette.base\n}\n"

	params := &protocol.ColorPresentationParams{
		Color: protocol.Color{
			Red:   0.1,
			Green: 0.09,
			Blue:  0.14,
			Alpha: 1.0,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 15},
			End:   protocol.Position{Line: 1, Character: 27},
		},
	}

	presentations := colorPresentation(content, params)

	if len(presentations) != 0 {
		t.Errorf("expected 0 presentations for palette reference, got %d", len(presentations))
	}
}

func TestColorPresentation_HashWithoutQuotes(t *testing.T) {
	// Test with content where the range text starts with # (no quotes)
	// This can happen if the range is just the hash+hex part
	content := "ansi {\n  red = #ff0000\n}\n"

	params := &protocol.ColorPresentationParams{
		Color: protocol.Color{
			Red:   0.0,
			Green: 1.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 8},
			End:   protocol.Position{Line: 1, Character: 15},
		},
	}

	presentations := colorPresentation(content, params)

	if len(presentations) != 1 {
		t.Fatalf("expected 1 presentation for # hex literal, got %d", len(presentations))
	}

	if presentations[0].Label != "#00ff00" {
		t.Errorf("expected label '#00ff00', got %q", presentations[0].Label)
	}

	if presentations[0].TextEdit == nil {
		t.Fatal("expected non-nil TextEdit")
	}

	// For bare # literal, TextEdit should replace with just #hex (no quotes)
	if presentations[0].TextEdit.NewText != "#00ff00" {
		t.Errorf("expected TextEdit.NewText '#00ff00', got %q", presentations[0].TextEdit.NewText)
	}
}

func TestColorPresentation_Integration(t *testing.T) {
	// Use the analyzer to produce real color locations, then test color presentation
	content := `palette {
  base = "#191724"
  love = "#eb6f92"
}

theme {
  background = palette.base
  cursor     = "#ff0000"
}
`
	result := Analyze("test.pstheme", content)

	// Get document colors
	infos := documentColors(result)
	if len(infos) == 0 {
		t.Fatal("expected at least one ColorInformation from analysis")
	}

	// Test presentation for each color location
	for i, cl := range result.Colors {
		params := &protocol.ColorPresentationParams{
			Color: infos[i].Color,
			Range: infos[i].Range,
		}

		presentations := colorPresentation(content, params)

		if cl.IsRef {
			// Palette references should not produce presentations
			if len(presentations) != 0 {
				t.Errorf("color %d (ref=%v): expected 0 presentations, got %d", i, cl.IsRef, len(presentations))
			}
		} else {
			// Hex literals should produce a presentation
			if len(presentations) != 1 {
				t.Errorf("color %d (ref=%v): expected 1 presentation, got %d", i, cl.IsRef, len(presentations))
			}
		}
	}
}
