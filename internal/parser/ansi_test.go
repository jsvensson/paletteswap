package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateANSI_Complete(t *testing.T) {
	theme := `
meta {
  name = "Test"
}

palette {
  base = "#191724"
}

theme {
  background = palette.base
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
	tmpFile := writeThemeFile(t, theme)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("complete ANSI should not error: %v", err)
	}
}

func TestValidateANSI_Missing(t *testing.T) {
	theme := `
meta {
  name = "Test"
}

palette {
  base = "#191724"
}

theme {
  background = palette.base
}

ansi {
  black = "#000000"
  red   = "#ff0000"
}
`
	tmpFile := writeThemeFile(t, theme)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for incomplete ANSI, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "ansi block incomplete") {
		t.Errorf("expected 'ansi block incomplete' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "green") {
		t.Errorf("expected missing color 'green' in error, got: %s", errMsg)
	}
}

func TestValidateANSI_NoBlock(t *testing.T) {
	theme := `
meta {
  name = "Test"
}

palette {
  base = "#191724"
}

theme {
  background = palette.base
}
`
	tmpFile := writeThemeFile(t, theme)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for missing ANSI block, got nil")
	}

	if !strings.Contains(err.Error(), "ansi block") {
		t.Errorf("expected 'ansi block' in error, got: %s", err.Error())
	}
}

func writeThemeFile(t *testing.T, content string) string {
	tmpFile := filepath.Join(t.TempDir(), "theme.hcl")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write theme file: %v", err)
	}
	return tmpFile
}
