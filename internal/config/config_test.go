package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
)

const sampleHCL = `
meta {
  name       = "Rose Pine"
  author     = "Test Author"
  appearance = "dark"
  url        = "https://example.com/theme"
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
  keyword  = palette.pine
  string   = palette.gold
  comment {
    color  = palette.surface
    italic = true
  }
  markup {
    heading = palette.love
    bold    = palette.gold
  }
}

ansi {
  black = palette.base
  red   = palette.love
}
`

func writeTempHCL(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.hcl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMeta(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if theme.Meta.Name != "Rose Pine" {
		t.Errorf("Meta.Name = %q, want %q", theme.Meta.Name, "Rose Pine")
	}
	if theme.Meta.Author != "Test Author" {
		t.Errorf("Meta.Author = %q, want %q", theme.Meta.Author, "Test Author")
	}
	if theme.Meta.Appearance != "dark" {
		t.Errorf("Meta.Appearance = %q, want %q", theme.Meta.Appearance, "dark")
	}
	if theme.Meta.URL != "https://example.com/theme" {
		t.Errorf("Meta.URL = %q, want %q", theme.Meta.URL, "https://example.com/theme")
	}
}

func TestLoadPalette(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(theme.Palette) != 6 {
		t.Errorf("len(Palette) = %d, want 6", len(theme.Palette))
	}
	love := theme.Palette["love"].(color.Style)
	if love.Color.Hex() != "#eb6f92" {
		t.Errorf("Palette[love].Color.Hex() = %q, want %q", love.Color.Hex(), "#eb6f92")
	}
}

func TestLoadTheme(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#191724" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#191724")
	}
	cursor := theme.Theme["cursor"]
	if cursor.Hex() != "#eb6f92" {
		t.Errorf("Theme[cursor].Hex() = %q, want %q", cursor.Hex(), "#eb6f92")
	}
}

func TestLoadSyntax(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Top-level syntax attribute (plain color becomes Style with no style flags)
	kw, ok := theme.Syntax["keyword"].(color.Style)
	if !ok {
		t.Fatal("Syntax[keyword] is not a Style")
	}
	if kw.Color.Hex() != "#31748f" {
		t.Errorf("Syntax[keyword].Color.Hex() = %q, want %q", kw.Color.Hex(), "#31748f")
	}
	if kw.Bold || kw.Italic || kw.Underline {
		t.Error("Syntax[keyword] should have no style flags set")
	}

	// Style block (comment with italic)
	comment, ok := theme.Syntax["comment"].(color.Style)
	if !ok {
		t.Fatal("Syntax[comment] is not a Style")
	}
	if comment.Color.Hex() != "#1f1d2e" {
		t.Errorf("Syntax[comment].Color.Hex() = %q, want %q", comment.Color.Hex(), "#1f1d2e")
	}
	if !comment.Italic {
		t.Error("Syntax[comment].Italic should be true")
	}
	if comment.Bold || comment.Underline {
		t.Error("Syntax[comment] should only have Italic set")
	}

	// Nested syntax block
	markup, ok := theme.Syntax["markup"].(color.ColorTree)
	if !ok {
		t.Fatal("Syntax[markup] is not a ColorTree")
	}
	heading, ok := markup["heading"].(color.Style)
	if !ok {
		t.Fatal("Syntax[markup][heading] is not a Style")
	}
	if heading.Color.Hex() != "#eb6f92" {
		t.Errorf("Syntax[markup][heading].Color.Hex() = %q, want %q", heading.Color.Hex(), "#eb6f92")
	}
}

func TestLoadANSI(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	black := theme.ANSI["black"]
	if black.Hex() != "#191724" {
		t.Errorf("ANSI[black].Hex() = %q, want %q", black.Hex(), "#191724")
	}
	red := theme.ANSI["red"]
	if red.Hex() != "#eb6f92" {
		t.Errorf("ANSI[red].Hex() = %q, want %q", red.Hex(), "#eb6f92")
	}
}

func TestLoadMissingPalette(t *testing.T) {
	hcl := `
meta {
  name = "test"
}
`
	path := writeTempHCL(t, hcl)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing palette block")
	}
}

func TestLoadInvalidHex(t *testing.T) {
	hcl := `
palette {
  bad = "not-a-color"
}
`
	path := writeTempHCL(t, hcl)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
}

func TestLoadStyleAllBools(t *testing.T) {
	hcl := `
palette {
  love = "#eb6f92"
}
syntax {
  keyword {
    color     = palette.love
    bold      = true
    italic    = true
    underline = true
  }
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	kw, ok := theme.Syntax["keyword"].(color.Style)
	if !ok {
		t.Fatal("Syntax[keyword] is not a Style")
	}
	if kw.Color.Hex() != "#eb6f92" {
		t.Errorf("Color.Hex() = %q, want %q", kw.Color.Hex(), "#eb6f92")
	}
	if !kw.Bold {
		t.Error("Bold should be true")
	}
	if !kw.Italic {
		t.Error("Italic should be true")
	}
	if !kw.Underline {
		t.Error("Underline should be true")
	}
}

func TestLoadStylePartial(t *testing.T) {
	hcl := `
palette {
  foam = "#9ccfd8"
}
syntax {
  link {
    color     = palette.foam
    underline = true
  }
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	link, ok := theme.Syntax["link"].(color.Style)
	if !ok {
		t.Fatal("Syntax[link] is not a Style")
	}
	if !link.Underline {
		t.Error("Underline should be true")
	}
	if link.Bold || link.Italic {
		t.Error("Bold and Italic should be false")
	}
}

func TestLoadSyntaxNestedStyleBlock(t *testing.T) {
	hcl := `
palette {
  gold = "#f6c177"
  iris = "#c4a7e7"
}
syntax {
  markup {
    bold {
      color = palette.gold
      bold  = true
    }
    italic {
      color  = palette.iris
      italic = true
    }
  }
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	markup, ok := theme.Syntax["markup"].(color.ColorTree)
	if !ok {
		t.Fatal("Syntax[markup] is not a ColorTree")
	}
	bold, ok := markup["bold"].(color.Style)
	if !ok {
		t.Fatal("markup[bold] is not a Style")
	}
	if !bold.Bold {
		t.Error("markup.bold.Bold should be true")
	}
	if bold.Color.Hex() != "#f6c177" {
		t.Errorf("markup.bold.Color.Hex() = %q, want %q", bold.Color.Hex(), "#f6c177")
	}
	italic, ok := markup["italic"].(color.Style)
	if !ok {
		t.Fatal("markup[italic] is not a Style")
	}
	if !italic.Italic {
		t.Error("markup.italic.Italic should be true")
	}
}

func TestLoadStyleMissingColor(t *testing.T) {
	// A block without "color" is treated as a nested scope, not a style block.
	// Parsing "bold = true" as a hex color string will panic on AsString().
	// This verifies the block is not silently accepted.
	hcl := `
palette {
  love = "#eb6f92"
}
syntax {
  keyword {
    bold = true
  }
}
`
	path := writeTempHCL(t, hcl)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when style block is missing color attribute")
		}
	}()
	_, _ = Load(path)
}

func TestLoadStyleUnknownAttribute(t *testing.T) {
	// Typos and unknown attributes in style blocks should produce an error.
	hcl := `
palette {
  love = "#eb6f92"
}
syntax {
  keyword {
    color = palette.love
    boldd = true
  }
}
`
	path := writeTempHCL(t, hcl)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown attribute 'boldd'")
	}
	if !strings.Contains(err.Error(), "unknown attribute") {
		t.Errorf("error should mention 'unknown attribute', got: %v", err)
	}
}

func TestLoadNestedPalette(t *testing.T) {
	hcl := `
palette {
  base = "#191724"

  highlight {
    low  = "#21202e"
    mid  = "#403d52"
    high = "#524f67"
  }

  custom {
    bold {
      color = "#ff0000"
      bold  = true
    }
  }
}

theme {
  background = palette.base
  cursor     = palette.highlight.high
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Check direct color
	base := theme.Palette["base"].(color.Style)
	if base.Color.Hex() != "#191724" {
		t.Errorf("Palette[base].Color.Hex() = %q, want %q", base.Color.Hex(), "#191724")
	}

	// Check nested color
	highlight, ok := theme.Palette["highlight"].(color.ColorTree)
	if !ok {
		t.Fatal("Palette[highlight] is not a ColorTree")
	}
	low := highlight["low"].(color.Style)
	if low.Color.Hex() != "#21202e" {
		t.Errorf("Palette[highlight][low].Color.Hex() = %q, want %q", low.Color.Hex(), "#21202e")
	}
	high := highlight["high"].(color.Style)
	if high.Color.Hex() != "#524f67" {
		t.Errorf("Palette[highlight][high].Color.Hex() = %q, want %q", high.Color.Hex(), "#524f67")
	}

	// Check nested style block
	custom, ok := theme.Palette["custom"].(color.ColorTree)
	if !ok {
		t.Fatal("Palette[custom] is not a ColorTree")
	}
	bold := custom["bold"].(color.Style)
	if bold.Color.Hex() != "#ff0000" {
		t.Errorf("Palette[custom][bold].Color.Hex() = %q, want %q", bold.Color.Hex(), "#ff0000")
	}
	if !bold.Bold {
		t.Error("Palette[custom][bold].Bold should be true")
	}

	// Check theme can reference nested palette values
	cursor := theme.Theme["cursor"]
	if cursor.Hex() != "#524f67" {
		t.Errorf("Theme[cursor].Hex() = %q, want %q", cursor.Hex(), "#524f67")
	}
}

func TestBrightenInTheme(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten(palette.base, 0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}

func TestBrightenWithLiteralHex(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten("#000000", 0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}
