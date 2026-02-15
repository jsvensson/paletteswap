package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
)

const completeANSI = `
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
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(theme.Palette.Children) != 6 {
		t.Errorf("len(Palette.Children) = %d, want 6", len(theme.Palette.Children))
	}
	love, err := theme.Palette.Lookup([]string{"love"})
	if err != nil {
		t.Fatalf("Lookup(love) error: %v", err)
	}
	if love.Hex() != "#eb6f92" {
		t.Errorf("Palette[love].Hex() = %q, want %q", love.Hex(), "#eb6f92")
	}
}

func TestLoadTheme(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
	markup, ok := theme.Syntax["markup"].(color.Tree)
	if !ok {
		t.Fatal("Syntax[markup] is not a Tree")
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
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
	_, err := Parse(path)
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
	_, err := Parse(path)
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	markup, ok := theme.Syntax["markup"].(color.Tree)
	if !ok {
		t.Fatal("Syntax[markup] is not a Tree")
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when style block is missing color attribute")
		}
	}()
	_, _ = Parse(path)
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
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
}

theme {
  background = palette.base
  cursor     = palette.highlight.high
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Check direct color
	base, err := theme.Palette.Lookup([]string{"base"})
	if err != nil {
		t.Fatalf("Lookup(base) error: %v", err)
	}
	if base.Hex() != "#191724" {
		t.Errorf("Palette[base].Hex() = %q, want %q", base.Hex(), "#191724")
	}

	// Check nested colors
	low, err := theme.Palette.Lookup([]string{"highlight", "low"})
	if err != nil {
		t.Fatalf("Lookup(highlight.low) error: %v", err)
	}
	if low.Hex() != "#21202e" {
		t.Errorf("Palette[highlight][low].Hex() = %q, want %q", low.Hex(), "#21202e")
	}
	high, err := theme.Palette.Lookup([]string{"highlight", "high"})
	if err != nil {
		t.Fatalf("Lookup(highlight.high) error: %v", err)
	}
	if high.Hex() != "#524f67" {
		t.Errorf("Palette[highlight][high].Hex() = %q, want %q", high.Hex(), "#524f67")
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
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
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}

func TestBrightenNegative(t *testing.T) {
	hcl := `
palette {
  white = "#ffffff"
}

theme {
  background = brighten(palette.white, -0.5)
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}

func TestBrightenInANSI(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

ansi {
  black   = brighten(palette.base, 0.5)
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
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	black := theme.ANSI["black"]
	if black.Hex() != "#7f7f7f" {
		t.Errorf("ANSI[black].Hex() = %q, want %q", black.Hex(), "#7f7f7f")
	}
}

func TestBrightenInSyntax(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

syntax {
  keyword = brighten(palette.base, 0.5)
  comment {
    color  = brighten(palette.base, 0.25)
    italic = true
  }
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	kw := theme.Syntax["keyword"].(color.Style)
	if kw.Color.Hex() != "#7f7f7f" {
		t.Errorf("Syntax[keyword].Color.Hex() = %q, want %q", kw.Color.Hex(), "#7f7f7f")
	}
	comment := theme.Syntax["comment"].(color.Style)
	if comment.Color.Hex() != "#3f3f3f" {
		t.Errorf("Syntax[comment].Color.Hex() = %q, want %q", comment.Color.Hex(), "#3f3f3f")
	}
}

func TestBrightenInvalidColor(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten("not-a-color", 0.5)
}
`
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for invalid color in brighten()")
	}
}

func TestDarkenInTheme(t *testing.T) {
	hcl := `
palette {
  white = "#ffffff"
}

theme {
  background = darken(palette.white, 0.5)
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}

func TestDarkenWithLiteralHex(t *testing.T) {
	hcl := `
palette {
  white = "#ffffff"
}

theme {
  background = darken("#ffffff", 0.5)
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}

func TestDarkenInvalidColor(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = darken("not-a-color", 0.5)
}
`
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for invalid color in darken()")
	}
}

func TestPaletteNestedColor(t *testing.T) {
	hcl := `
palette {
  gray = "#c0c0c0"

  highlight {
    color = palette.gray
    low   = "#21202e"
    high  = "#524f67"
  }
}

theme {
  background = palette.highlight
  surface    = palette.highlight.low
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	highlightColor, err := theme.Palette.Lookup([]string{"highlight"})
	if err != nil {
		t.Fatalf("Lookup(highlight) error: %v", err)
	}
	if highlightColor.Hex() != "#c0c0c0" {
		t.Errorf("palette.highlight = %q, want %q", highlightColor.Hex(), "#c0c0c0")
	}

	lowColor, err := theme.Palette.Lookup([]string{"highlight", "low"})
	if err != nil {
		t.Fatalf("Lookup(highlight.low) error: %v", err)
	}
	if lowColor.Hex() != "#21202e" {
		t.Errorf("palette.highlight.low = %q, want %q", lowColor.Hex(), "#21202e")
	}

	bg := theme.Theme["background"]
	if bg.Hex() != "#c0c0c0" {
		t.Errorf("Theme[background] = %q, want %q", bg.Hex(), "#c0c0c0")
	}

	surface := theme.Theme["surface"]
	if surface.Hex() != "#21202e" {
		t.Errorf("Theme[surface] = %q, want %q", surface.Hex(), "#21202e")
	}
}

func TestPaletteDeepNesting(t *testing.T) {
	hcl := `
palette {
  highlight {
    color = "#c0c0c0"
    deep {
      color = "#100f1a"
      muted = "#0a0a10"
    }
  }
}

theme {
  a = palette.highlight
  b = palette.highlight.deep
  c = palette.highlight.deep.muted
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if theme.Theme["a"].Hex() != "#c0c0c0" {
		t.Errorf("a = %q, want %q", theme.Theme["a"].Hex(), "#c0c0c0")
	}
	if theme.Theme["b"].Hex() != "#100f1a" {
		t.Errorf("b = %q, want %q", theme.Theme["b"].Hex(), "#100f1a")
	}
	if theme.Theme["c"].Hex() != "#0a0a10" {
		t.Errorf("c = %q, want %q", theme.Theme["c"].Hex(), "#0a0a10")
	}
}

func TestPaletteNamespaceOnlyError(t *testing.T) {
	hcl := `
palette {
  highlight {
    low = "#21202e"
  }
}

theme {
  background = palette.highlight
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error when referencing namespace-only block as color")
	}
}

func TestPaletteSelfReference(t *testing.T) {
	hcl := `
palette {
  base = "#191724"
  surface = palette.base
}

theme {
  background = palette.surface
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if theme.Theme["background"].Hex() != "#191724" {
		t.Errorf("background = %q, want %q", theme.Theme["background"].Hex(), "#191724")
	}
}

func TestPaletteTransformLightness(t *testing.T) {
	hcl := `
palette {
  base = "#808080"

  transform {
    lightness {
      range = [0.4, 0.8]
      steps = 3
    }
  }
}

theme {
  background = palette.base
  step1      = palette.base.l1
  step2      = palette.base.l2
  step3      = palette.base.l3
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Original color still accessible
	bg := theme.Theme["background"]
	if bg.Hex() != "#808080" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#808080")
	}

	// Stepped children exist and are referenceable
	s1 := theme.Theme["step1"]
	if s1.Hex() == "" {
		t.Fatal("palette.base.l1 should exist")
	}
	s2 := theme.Theme["step2"]
	if s2.Hex() == "" {
		t.Fatal("palette.base.l2 should exist")
	}
	s3 := theme.Theme["step3"]
	if s3.Hex() == "" {
		t.Fatal("palette.base.l3 should exist")
	}

	// l1, l2, l3 should be different from each other (different lightness)
	if s1.Hex() == s2.Hex() && s2.Hex() == s3.Hex() {
		t.Error("stepped children should have different colors")
	}
}

func TestPaletteTransformLightnessNested(t *testing.T) {
	hcl := `
palette {
  highlight {
    mid = "#808080"
  }

  transform {
    lightness {
      range = [0.3, 0.7]
      steps = 2
    }
  }
}

theme {
  a = palette.highlight.mid.l1
  b = palette.highlight.mid.l2
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	a := theme.Theme["a"]
	if a.Hex() == "" {
		t.Fatal("palette.highlight.mid.l1 should exist")
	}
	b := theme.Theme["b"]
	if b.Hex() == "" {
		t.Fatal("palette.highlight.mid.l2 should exist")
	}
	if a.Hex() == b.Hex() {
		t.Error("l1 and l2 should have different lightness values")
	}
}

func TestPaletteNoTransform(t *testing.T) {
	// Verify existing sampleHCL (no transform) still works, no stepped children
	path := writeTempHCL(t, sampleHCL)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// base should have no children (no transform applied)
	baseNode := theme.Palette.Children["base"]
	if baseNode == nil {
		t.Fatal("palette.base should exist")
	}
	if baseNode.Children != nil {
		t.Error("palette.base should have no children without transform block")
	}
}

func TestPaletteForwardReferenceError(t *testing.T) {
	hcl := `
palette {
  surface = palette.base
  base    = "#191724"
}

theme {
  background = palette.surface
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for forward reference in palette")
	}
}
