# Required ANSI Block + Universal Template Path API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ANSI validation requiring all 16 colors, and extend template functions to support universal dot-notation paths across all blocks.

**Architecture:** Add validation in config.Load(), replace template functions with path-based versions supporting "block.path" syntax, maintain backward compatibility for direct field access temporarily.

**Tech Stack:** Go 1.21+, hashicorp/hcl, text/template

---

## Task 1: Add ANSI Validation Constants and Tests

**Files:**
- Modify: `internal/config/config.go` (add constants near top)
- Create: `internal/config/ansi_test.go`

**Step 1: Write failing test for complete ANSI validation**

Create `internal/config/ansi_test.go`:

```go
package config

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestValidateANSI -v`
Expected: FAIL - validateANSI function does not exist

**Step 3: Add ANSI validation constants**

In `internal/config/config.go`, add after imports:

```go
// requiredANSIColors defines the 16 standard terminal colors that must be present.
var requiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}
```

**Step 4: Implement validateANSI function**

In `internal/config/config.go`, add before Load():

```go
// validateANSI checks that all 16 required ANSI colors are present.
func validateANSI(ansi map[string]color.Color) error {
	if len(ansi) == 0 {
		return fmt.Errorf("ansi block incomplete: no colors defined")
	}

	var missing []string
	for _, colorName := range requiredANSIColors {
		if _, ok := ansi[colorName]; !ok {
			missing = append(missing, colorName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("ansi block incomplete\nMissing colors: %s\nRequired colors: %s",
			strings.Join(missing, ", "),
			strings.Join(requiredANSIColors, ", "))
	}

	return nil
}
```

**Step 5: Call validateANSI in Load()**

In `internal/config/config.go`, update Load() function, add validation after parsing ANSI but before returning Theme:

```go
// After line 194 (after ansiColors parsing):
	if err := validateANSI(ansiColors); err != nil {
		return nil, err
	}
```

**Step 6: Add missing import**

At top of `internal/config/config.go`, ensure "strings" is imported:

```go
import (
	"fmt"
	"os"
	"sort"
	"strings"  // Add if not present
	// ... rest of imports
)
```

**Step 7: Run tests to verify they pass**

Run: `go test ./internal/config -run TestValidateANSI -v`
Expected: PASS (3 tests)

**Step 8: Commit**

```bash
git add internal/config/config.go internal/config/ansi_test.go
git commit -m "feat: add ANSI block validation requiring all 16 colors

- Add requiredANSIColors constant with 16 standard terminal colors
- Add validateANSI() function to check completeness
- Call validation in Load() after parsing ANSI block
- Add tests for complete, incomplete, and missing ANSI blocks

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Add Universal Path Resolution for Template Functions

**Files:**
- Modify: `internal/engine/engine.go`
- Create: `internal/engine/path_test.go`

**Step 1: Write failing test for path resolution**

Create `internal/engine/path_test.go`:

```go
package engine

import (
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
)

func TestResolveColorPath_Palette(t *testing.T) {
	data := templateData{
		Palette: color.ColorTree{
			"base": color.Style{Color: color.Color{R: 25, G: 23, B: 36}},
			"highlight": color.ColorTree{
				"low": color.Style{Color: color.Color{R: 33, G: 32, B: 46}},
			},
		},
	}

	tests := []struct {
		path string
		want color.Color
	}{
		{"palette.base", color.Color{R: 25, G: 23, B: 36}},
		{"palette.highlight.low", color.Color{R: 33, G: 32, B: 46}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := resolveColorPath(tt.path, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveColorPath_Theme(t *testing.T) {
	data := templateData{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
			"foreground": {R: 224, G: 222, B: 244},
		},
	}

	got, err := resolveColorPath("theme.background", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := color.Color{R: 25, G: 23, B: 36}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveColorPath_ANSI(t *testing.T) {
	data := templateData{
		ANSI: map[string]color.Color{
			"black":        {R: 0, G: 0, B: 0},
			"bright_black": {R: 128, G: 128, B: 128},
		},
	}

	tests := []struct {
		path string
		want color.Color
	}{
		{"ansi.black", color.Color{R: 0, G: 0, B: 0}},
		{"ansi.bright_black", color.Color{R: 128, G: 128, B: 128}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := resolveColorPath(tt.path, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveColorPath_Syntax(t *testing.T) {
	data := templateData{
		Syntax: color.ColorTree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
		},
	}

	got, err := resolveColorPath("syntax.keyword", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := color.Color{R: 49, G: 116, B: 143}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveColorPath_InvalidBlock(t *testing.T) {
	data := templateData{}

	_, err := resolveColorPath("invalid.path", data)
	if err == nil {
		t.Fatal("expected error for invalid block, got nil")
	}
}

func TestResolveColorPath_PathNotFound(t *testing.T) {
	data := templateData{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	_, err := resolveColorPath("theme.notfound", data)
	if err == nil {
		t.Fatal("expected error for path not found, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run TestResolveColorPath -v`
Expected: FAIL - resolveColorPath function does not exist

**Step 3: Implement resolveColorPath function**

In `internal/engine/engine.go`, add before buildTemplateData():

```go
// resolveColorPath resolves a universal dot-notation path to a Color.
// Supports paths like "palette.base", "theme.background", "ansi.black", "syntax.keyword".
func resolveColorPath(path string, data templateData) (color.Color, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return color.Color{}, fmt.Errorf("invalid path %q: must be block.name format", path)
	}

	block := parts[0]
	rest := parts[1:]

	switch block {
	case "palette":
		style := getStyleFromTree(data.Palette, rest)
		if style.Color == (color.Color{}) {
			return color.Color{}, fmt.Errorf("palette path not found: %s", path)
		}
		return style.Color, nil

	case "theme":
		if len(rest) != 1 {
			return color.Color{}, fmt.Errorf("theme paths must be single-level: %s", path)
		}
		c, ok := data.Theme[rest[0]]
		if !ok {
			return color.Color{}, fmt.Errorf("theme color not found: %s", rest[0])
		}
		return c, nil

	case "ansi":
		if len(rest) != 1 {
			return color.Color{}, fmt.Errorf("ansi paths must be single-level: %s", path)
		}
		c, ok := data.ANSI[rest[0]]
		if !ok {
			return color.Color{}, fmt.Errorf("ansi color not found: %s", rest[0])
		}
		return c, nil

	case "syntax":
		style := getStyleFromTree(data.Syntax, rest)
		if style.Color == (color.Color{}) {
			return color.Color{}, fmt.Errorf("syntax path not found: %s", path)
		}
		return style.Color, nil

	default:
		return color.Color{}, fmt.Errorf("unknown block %q (valid: palette, theme, ansi, syntax)", block)
	}
}

// getStyleFromTree traverses a ColorTree using path segments and returns the Style.
func getStyleFromTree(tree color.ColorTree, path []string) color.Style {
	if len(path) == 0 {
		return color.Style{}
	}

	current := tree
	for i, part := range path {
		val, ok := current[part]
		if !ok {
			return color.Style{}
		}

		// Last part should be a Style
		if i == len(path)-1 {
			if style, ok := val.(color.Style); ok {
				return style
			}
			return color.Style{}
		}

		// Intermediate parts should be ColorTrees
		if subtree, ok := val.(color.ColorTree); ok {
			current = subtree
		} else {
			return color.Style{}
		}
	}

	return color.Style{}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine -run TestResolveColorPath -v`
Expected: PASS (6 tests)

**Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/path_test.go
git commit -m "feat: add universal path resolution for template functions

- Add resolveColorPath() supporting palette/theme/ansi/syntax blocks
- Add getStyleFromTree() helper for nested ColorTree navigation
- Add comprehensive tests for all block types and error cases

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Replace Template Functions with Universal Path Support

**Files:**
- Modify: `internal/engine/engine.go`
- Create: `internal/engine/functions_test.go`

**Step 1: Write failing test for new template functions**

Create `internal/engine/functions_test.go`:

```go
package engine

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/config"
)

func TestTemplateFunctions_Hex(t *testing.T) {
	theme := &config.Theme{
		Palette: color.ColorTree{
			"base": color.Style{Color: color.Color{R: 25, G: 23, B: 36}},
		},
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
		ANSI: map[string]color.Color{
			"black": {R: 0, G: 0, B: 0},
		},
		Syntax: color.ColorTree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
		},
	}

	data := buildTemplateData(theme)

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"palette path", `{{ hex "palette.base" }}`, "#191724"},
		{"theme path", `{{ hex "theme.background" }}`, "#191724"},
		{"ansi path", `{{ hex "ansi.black" }}`, "#000000"},
		{"syntax path", `{{ hex "syntax.keyword" }}`, "#31748f"},
		{"direct field", `{{ hex .Theme.background }}`, "#191724"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute error: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateFunctions_Bhex(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ bhex "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "191724"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Hexa(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ hexa "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "#191724ff"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Bhexa(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ bhexa "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "191724ff"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_RGB(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tests := []struct {
		template string
		want     string
	}{
		{`{{ rgb "theme.background" }}`, "rgb(25, 23, 36)"},
		{`{{ rgb .Theme.background }}`, "rgb(25, 23, 36)"},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute error: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateFunctions_RGBA(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ rgba "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "rgba(25, 23, 36, 1.0)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Style(t *testing.T) {
	theme := &config.Theme{
		Syntax: color.ColorTree{
			"keyword": color.Style{
				Color: color.Color{R: 49, G: 116, B: 143},
				Bold:  true,
			},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(
		`{{ $s := style "syntax.keyword" }}{{ if $s.Bold }}bold{{ end }}`,
	)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "bold"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run TestTemplateFunctions -v`
Expected: FAIL - new functions not registered

**Step 3: Add alpha formatting methods to color.Color**

In `internal/color/color.go`, add after RGB() method:

```go
// Hexa returns the color as a hex string with alpha channel, e.g. "#eb6f92ff".
// Alpha is hardcoded to full opacity (ff) for now.
func (c Color) Hexa() string {
	return fmt.Sprintf("#%02x%02x%02xff", c.R, c.G, c.B)
}

// BareHexa returns the color as a hex string with alpha but no hash, e.g. "eb6f92ff".
// Alpha is hardcoded to full opacity (ff) for now.
func (c Color) BareHexa() string {
	return fmt.Sprintf("%02x%02x%02xff", c.R, c.G, c.B)
}

// RGBA returns the color as an rgba() string with full opacity, e.g. "rgba(235, 111, 146, 1.0)".
func (c Color) RGBA() string {
	return fmt.Sprintf("rgba(%d, %d, %d, 1.0)", c.R, c.G, c.B)
}
```

**Step 4: Replace buildTemplateData FuncMap**

In `internal/engine/engine.go`, replace the buildTemplateData function (lines 94-119):

```go
func buildTemplateData(theme *config.Theme) templateData {
	data := templateData{
		Meta:    theme.Meta,
		Palette: theme.Palette,
		Theme:   theme.Theme,
		Syntax:  theme.Syntax,
		ANSI:    theme.ANSI,
	}

	// Universal path-based functions
	data.FuncMap = template.FuncMap{
		"hex": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.Hex(), nil
			case color.Color:
				return v.Hex(), nil
			default:
				return "", fmt.Errorf("hex: unsupported type %T", arg)
			}
		},
		"bhex": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.HexBare(), nil
			case color.Color:
				return v.HexBare(), nil
			default:
				return "", fmt.Errorf("bhex: unsupported type %T", arg)
			}
		},
		"hexa": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.Hexa(), nil
			case color.Color:
				return v.Hexa(), nil
			default:
				return "", fmt.Errorf("hexa: unsupported type %T", arg)
			}
		},
		"bhexa": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.BareHexa(), nil
			case color.Color:
				return v.BareHexa(), nil
			default:
				return "", fmt.Errorf("bhexa: unsupported type %T", arg)
			}
		},
		"rgb": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.RGB(), nil
			case color.Color:
				return v.RGB(), nil
			default:
				return "", fmt.Errorf("rgb: unsupported type %T", arg)
			}
		},
		"rgba": func(arg interface{}) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.RGBA(), nil
			case color.Color:
				return v.RGBA(), nil
			default:
				return "", fmt.Errorf("rgba: unsupported type %T", arg)
			}
		},
		"style": func(path string) (color.Style, error) {
			parts := strings.Split(path, ".")
			if len(parts) < 2 {
				return color.Style{}, fmt.Errorf("invalid path %q", path)
			}

			block := parts[0]
			rest := parts[1:]

			switch block {
			case "palette":
				return getStyleFromTree(data.Palette, rest), nil
			case "syntax":
				return getStyleFromTree(data.Syntax, rest), nil
			default:
				return color.Style{}, fmt.Errorf("style only supports palette/syntax blocks, got %q", block)
			}
		},
	}

	return data
}
```

**Step 5: Remove old getStyleFromPath function**

In `internal/engine/engine.go`, delete the old getStyleFromPath function (lines 121-150, after the new buildTemplateData).

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/engine -v`
Expected: PASS (all tests including new function tests)

**Step 7: Commit**

```bash
git add internal/engine/engine.go internal/engine/functions_test.go internal/color/color.go
git commit -m "feat: replace template functions with universal path API

- Replace old hex/hexBare/rgb with hex/bhex/rgb supporting paths
- Add hexa/bhexa/rgba functions with hardcoded alpha
- Support both path strings and direct Color field access
- Update style function to use path-based API
- Add Hexa/BareHexa/RGBA methods to color.Color
- Remove old palette function and getStyleFromPath

Breaking: hexBare renamed to bhex, palette function removed

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Update Zed Template to Use New API

**Files:**
- Modify: `templates/zed.json.tmpl`

**Step 1: Update theme and ANSI color references**

In `templates/zed.json.tmpl`, replace direct field access with path notation:

Change lines 10-37 from:
```
        "background": "{{ hex .Theme.background }}",
        "editor.background": "{{ hex .Theme.background }}",
        ...
        "terminal.ansi.black": "{{ hex .ANSI.black }}",
```

To:
```
        "background": "{{ hex "theme.background" }}",
        "editor.background": "{{ hex "theme.background" }}",
        "editor.foreground": "{{ hex "theme.foreground" }}",
        "editor.gutter.background": "{{ hex "theme.background" }}",
        "editor.active_line.background": "{{ hex "theme.selection" }}",
        "border": "{{ hex "theme.border" }}",
        "border.variant": "{{ hex "theme.active_border" }}",
        "tab_bar.background": "{{ hex "theme.background" }}",
        "tab.inactive_background": "{{ hex "theme.inactive_tab" }}",
        "tab.active_background": "{{ hex "theme.active_tab" }}",
        "terminal.background": "{{ hex "theme.background" }}",
        "terminal.foreground": "{{ hex "theme.foreground" }}",
        "terminal.ansi.black": "{{ hex "ansi.black" }}",
        "terminal.ansi.red": "{{ hex "ansi.red" }}",
        "terminal.ansi.green": "{{ hex "ansi.green" }}",
        "terminal.ansi.yellow": "{{ hex "ansi.yellow" }}",
        "terminal.ansi.blue": "{{ hex "ansi.blue" }}",
        "terminal.ansi.magenta": "{{ hex "ansi.magenta" }}",
        "terminal.ansi.cyan": "{{ hex "ansi.cyan" }}",
        "terminal.ansi.white": "{{ hex "ansi.white" }}",
        "terminal.ansi.bright_black": "{{ hex "ansi.bright_black" }}",
        "terminal.ansi.bright_red": "{{ hex "ansi.bright_red" }}",
        "terminal.ansi.bright_green": "{{ hex "ansi.bright_green" }}",
        "terminal.ansi.bright_yellow": "{{ hex "ansi.bright_yellow" }}",
        "terminal.ansi.bright_blue": "{{ hex "ansi.bright_blue" }}",
        "terminal.ansi.bright_magenta": "{{ hex "ansi.bright_magenta" }}",
        "terminal.ansi.bright_cyan": "{{ hex "ansi.bright_cyan" }}",
        "terminal.ansi.bright_white": "{{ hex "ansi.bright_white" }}",
```

**Step 2: Update syntax color references**

In `templates/zed.json.tmpl`, change the syntax section (lines 39-122), replace each instance of:
```
            "color": "{{ .Color | hex }}"
```

With:
```
            "color": "{{ hex "syntax.keyword" }}"
            "color": "{{ hex "syntax.string" }}"
            "color": "{{ hex "syntax.comment" }}"
            // etc for each syntax element
```

Full replacement for lines 38-123:

```
        "syntax": {
          {{- with .Syntax.keyword }}
          "keyword": {
            "color": "{{ hex "syntax.keyword" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.string }}
          "string": {
            "color": "{{ hex "syntax.string" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.comment }}
          "comment": {
            "color": "{{ hex "syntax.comment" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.variable }}
          "variable": {
            "color": "{{ hex "syntax.variable" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.function }}
          "function": {
            "color": "{{ hex "syntax.function" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.type }}
          "type": {
            "color": "{{ hex "syntax.type" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.constant }}
          "constant": {
            "color": "{{ hex "syntax.constant" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.number }}
          "number": {
            "color": "{{ hex "syntax.number" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.boolean }}
          "boolean": {
            "color": "{{ hex "syntax.boolean" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.property }}
          "property": {
            "color": "{{ hex "syntax.property" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.tag }}
          "tag": {
            "color": "{{ hex "syntax.tag" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          },
          {{- end }}
          {{- with .Syntax.attribute }}
          "attribute": {
            "color": "{{ hex "syntax.attribute" }}"
            {{- if .Bold }}, "font_weight": 700{{ end }}
            {{- if .Italic }}, "font_style": "italic"{{ end }}
          }
          {{- end }}
        },
```

**Step 3: Update link_text.accent**

Change line 124 from:
```
        "link_text.accent": "{{ hex .Theme.url }}"
```

To:
```
        "link_text.accent": "{{ hex "theme.url" }}"
```

**Step 4: Verify template syntax**

Run: `go run cmd/paletteswap/main.go generate --help`
Expected: Command help output (validates template parses)

**Step 5: Commit**

```bash
git add templates/zed.json.tmpl
git commit -m "feat: update Zed template to use universal path API

- Replace direct field access with path notation
- Use hex \"theme.background\" instead of hex .Theme.background
- Use hex \"ansi.black\" instead of hex .ANSI.black
- Use hex \"syntax.keyword\" instead of .Color | hex

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Update Example Theme Files with Complete ANSI

**Files:**
- Find and modify: example theme files (likely in examples/ or testdata/)

**Step 1: Find example theme files**

Run: `find . -name "*.hcl" -type f`
Expected: List of HCL theme files

**Step 2: Check each theme for ANSI completeness**

For each file found, verify it has all 16 ANSI colors. If missing, add them.

**Step 3: Test with actual generation**

Run: `go run cmd/paletteswap/main.go generate --theme <path>`
Expected: Success (validates ANSI validation works)

**Step 4: Commit any updated theme files**

```bash
git add <theme-files>
git commit -m "feat: add complete ANSI blocks to example themes

- Ensure all example themes have 16 required ANSI colors
- Validates ANSI validation requirement

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Run All Tests and Verify

**Step 1: Run all tests**

Run: `go test ./...`
Expected: PASS (all packages)

**Step 2: Build the binary**

Run: `go build -o paletteswap cmd/paletteswap/main.go`
Expected: Success, binary created

**Step 3: Test with real theme generation**

Run: `./paletteswap generate` (using default or test theme)
Expected: Success, themes generated

**Step 4: Verify output**

Check generated theme files for correct formatting using new API.

**Step 5: Clean up if needed**

Run: `go mod tidy`

---

## Notes for Implementer

**ANSI validation:**
- Validates during Load(), before template execution
- Clear error messages listing missing colors
- Blocks theme loading if incomplete

**Universal path API:**
- Supports "block.path" notation for all blocks
- Backward compatible with direct field access (temporary)
- Path resolution handles nested ColorTrees (palette, syntax)
- Maps handle flat access (theme, ansi)

**Alpha channel:**
- Hardcoded to full opacity (ff / 1.0) for now
- Future: configurable alpha per color

**Breaking changes:**
- `hexBare` → `bhex`
- `palette "path" | hex` → `hex "palette.path"`
- ANSI block now required with all 16 colors

**Testing strategy:**
- Unit tests for each component (validation, path resolution, functions)
- Integration tests via template execution
- Real-world validation with theme generation
