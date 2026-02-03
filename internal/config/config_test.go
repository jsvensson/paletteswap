package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
)

const sampleHCL = `
meta {
  name       = "Rose Pine"
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
  keyword  = palette.pine
  string   = palette.gold
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
	love := theme.Palette["love"]
	if love.Hex() != "#eb6f92" {
		t.Errorf("Palette[love].Hex() = %q, want %q", love.Hex(), "#eb6f92")
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

	// Top-level syntax attribute
	kw, ok := theme.Syntax["keyword"].(color.Color)
	if !ok {
		t.Fatal("Syntax[keyword] is not a Color")
	}
	if kw.Hex() != "#31748f" {
		t.Errorf("Syntax[keyword].Hex() = %q, want %q", kw.Hex(), "#31748f")
	}

	// Nested syntax block
	markup, ok := theme.Syntax["markup"].(color.ColorTree)
	if !ok {
		t.Fatal("Syntax[markup] is not a ColorTree")
	}
	heading, ok := markup["heading"].(color.Color)
	if !ok {
		t.Fatal("Syntax[markup][heading] is not a Color")
	}
	if heading.Hex() != "#eb6f92" {
		t.Errorf("Syntax[markup][heading].Hex() = %q, want %q", heading.Hex(), "#eb6f92")
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
