# Template API Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the vague `palette` function with piping syntax with direct, precisely-named color formatting functions.

**Architecture:** Remove the `palette` function and convert existing filter functions (`hex`, `hexBare`, `rgb`) from taking `color.Color` to taking `string` paths. Add new alpha-channel variants (`hexa`, `bhexa`, `rgba`) with hardcoded full opacity.

**Tech Stack:** Go 1.x, text/template, internal color package

---

## Task 1: Add RGBA Method to Color Type

**Files:**
- Modify: `internal/color/color.go`
- Test: `internal/color/color_test.go`

**Step 1: Write the failing test**

Add to `internal/color/color_test.go`:

```go
func TestColor_RGBA(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected string
	}{
		{
			name:     "red with full opacity",
			hex:      "#ff0000",
			expected: "rgba(255, 0, 0, 1.0)",
		},
		{
			name:     "green with full opacity",
			hex:      "#00ff00",
			expected: "rgba(0, 255, 0, 1.0)",
		},
		{
			name:     "dark color",
			hex:      "#191724",
			expected: "rgba(25, 23, 36, 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MustParse(tt.hex)
			got := c.RGBA()
			if got != tt.expected {
				t.Errorf("RGBA() = %v, want %v", got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/color -v -run TestColor_RGBA`
Expected: FAIL with "undefined: Color.RGBA"

**Step 3: Write minimal implementation**

Add to `internal/color/color.go` after the `RGB()` method:

```go
// RGBA returns the color in rgba() function format with full opacity
func (c Color) RGBA() string {
	r, g, b := c.RGB255()
	return fmt.Sprintf("rgba(%d, %d, %d, 1.0)", r, g, b)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/color -v -run TestColor_RGBA`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/color/color.go internal/color/color_test.go
git commit -m "feat: add Color.RGBA() method for alpha channel support"
```

---

## Task 2: Add HexAlpha and HexBareAlpha Methods to Color Type

**Files:**
- Modify: `internal/color/color.go`
- Test: `internal/color/color_test.go`

**Step 1: Write the failing test**

Add to `internal/color/color_test.go`:

```go
func TestColor_HexAlpha(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected string
	}{
		{
			name:     "red with full opacity",
			hex:      "#ff0000",
			expected: "#ff0000ff",
		},
		{
			name:     "dark color",
			hex:      "#191724",
			expected: "#191724ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MustParse(tt.hex)
			got := c.HexAlpha()
			if got != tt.expected {
				t.Errorf("HexAlpha() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestColor_HexBareAlpha(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected string
	}{
		{
			name:     "red with full opacity",
			hex:      "#ff0000",
			expected: "ff0000ff",
		},
		{
			name:     "dark color",
			hex:      "#191724",
			expected: "191724ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MustParse(tt.hex)
			got := c.HexBareAlpha()
			if got != tt.expected {
				t.Errorf("HexBareAlpha() = %v, want %v", got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/color -v -run "TestColor_HexAlpha|TestColor_HexBareAlpha"`
Expected: FAIL with "undefined: Color.HexAlpha" and "undefined: Color.HexBareAlpha"

**Step 3: Write minimal implementation**

Add to `internal/color/color.go` after the `HexBare()` method:

```go
// HexAlpha returns the color in hex format with alpha channel (#rrggbbaa)
func (c Color) HexAlpha() string {
	return c.Hex() + "ff"
}

// HexBareAlpha returns the color in hex format without # prefix and with alpha channel (rrggbbaa)
func (c Color) HexBareAlpha() string {
	return c.HexBare() + "ff"
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/color -v -run "TestColor_HexAlpha|TestColor_HexBareAlpha"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/color/color.go internal/color/color_test.go
git commit -m "feat: add HexAlpha() and HexBareAlpha() methods for alpha channel"
```

---

## Task 3: Update Template Functions to New API

**Files:**
- Modify: `internal/engine/engine.go:101-117`
- Test: `internal/engine/engine_test.go`

**Step 1: Write the failing test**

Add to `internal/engine/engine_test.go`:

```go
func TestTemplateFunctions_NewAPI(t *testing.T) {
	theme := &config.Theme{
		Palette: config.ColorTree{
			"base": color.Style{Color: color.MustParse("#191724")},
			"highlight": config.ColorTree{
				"low": color.Style{Color: color.MustParse("#21202e")},
			},
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "hex function with top-level path",
			template: `{{ hex "base" }}`,
			expected: "#191724",
		},
		{
			name:     "hex function with nested path",
			template: `{{ hex "highlight.low" }}`,
			expected: "#21202e",
		},
		{
			name:     "bhex function",
			template: `{{ bhex "base" }}`,
			expected: "191724",
		},
		{
			name:     "hexa function",
			template: `{{ hexa "base" }}`,
			expected: "#191724ff",
		},
		{
			name:     "bhexa function",
			template: `{{ bhexa "base" }}`,
			expected: "191724ff",
		},
		{
			name:     "rgb function",
			template: `{{ rgb "base" }}`,
			expected: "rgb(25, 23, 36)",
		},
		{
			name:     "rgba function",
			template: `{{ rgba "base" }}`,
			expected: "rgba(25, 23, 36, 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildTemplateData(theme)
			tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatalf("template parse error: %v", err)
			}

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("template execute error: %v", err)
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -v -run TestTemplateFunctions_NewAPI`
Expected: FAIL with function errors or wrong output

**Step 3: Replace template function map**

In `internal/engine/engine.go`, replace the `FuncMap` in `buildTemplateData()` (lines 101-117) with:

```go
FuncMap: template.FuncMap{
	"hex": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.Hex(), nil
	},
	"bhex": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.HexBare(), nil
	},
	"hexa": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.HexAlpha(), nil
	},
	"bhexa": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.HexBareAlpha(), nil
	},
	"rgb": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.RGB(), nil
	},
	"rgba": func(path string) (string, error) {
		style, err := getStyleFromPathWithError(theme.Palette, path)
		if err != nil {
			return "", err
		}
		return style.Color.RGBA(), nil
	},
	"style": func(path string) color.Style {
		return getStyleFromPath(theme.Palette, path)
	},
},
```

**Step 4: Add error-returning helper function**

Add after the existing `getStyleFromPath()` function in `internal/engine/engine.go`:

```go
// getStyleFromPathWithError traverses a ColorTree using dot-separated path and returns error if not found
func getStyleFromPathWithError(tree config.ColorTree, path string) (color.Style, error) {
	parts := strings.Split(path, ".")
	current := tree

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return color.Style{}, fmt.Errorf("palette path not found: %s", path)
		}

		// If this is the last part, we expect a Style
		if i == len(parts)-1 {
			if style, ok := value.(color.Style); ok {
				return style, nil
			}
			return color.Style{}, fmt.Errorf("palette path %s is not a color", path)
		}

		// Otherwise, we expect a ColorTree to continue traversing
		if subtree, ok := value.(config.ColorTree); ok {
			current = subtree
		} else {
			return color.Style{}, fmt.Errorf("palette path %s: expected subtree at %s", path, part)
		}
	}

	return color.Style{}, fmt.Errorf("palette path not found: %s", path)
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/engine -v -run TestTemplateFunctions_NewAPI`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: replace palette function with direct color formatting functions"
```

---

## Task 4: Remove Old Palette Function Tests

**Files:**
- Modify: `internal/engine/engine_test.go`

**Step 1: Find and remove old tests**

Search for tests that use the old `palette` function with piping syntax and remove them. Look for patterns like:
- `{{ palette "..." | hex }}`
- `{{ palette "..." | hexBare }}`
- `{{ palette "..." | rgb }}`

Run: `grep -n "palette.*|" internal/engine/engine_test.go`

**Step 2: Delete or update those test cases**

If tests exist that use the old syntax, remove them or update them to use the new direct function syntax.

**Step 3: Run all engine tests**

Run: `go test ./internal/engine -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/engine/engine_test.go
git commit -m "test: remove old palette function tests"
```

---

## Task 5: Update Example Templates

**Files:**
- Modify: `templates/ghostty.tmpl`
- Modify: `templates/zed.json.tmpl`

**Step 1: Update Ghostty template**

Read the current template:
Run: `cat templates/ghostty.tmpl`

Replace all instances of `hexBare` with `bhex`:
- Find: `{{ hexBare`
- Replace: `{{ bhex`

**Step 2: Update Zed template**

Read the current template:
Run: `cat templates/zed.json.tmpl`

Replace all instances of `hex` filter usage with direct function calls. Look for patterns like:
- `{{ .Field | hex }}` → remains unchanged (this is field access)
- `{{ palette "..." | hex }}` → `{{ hex "..." }}`

**Step 3: Test template rendering**

Run: `go run cmd/paletteswap/main.go generate`
Expected: Templates generate successfully without errors

**Step 4: Verify output**

Check that generated files in `output/` directory have correct formatting.

**Step 5: Commit**

```bash
git add templates/ghostty.tmpl templates/zed.json.tmpl
git commit -m "feat: update templates to use new API"
```

---

## Task 6: Update README Documentation

**Files:**
- Modify: `README.md:159-165` (Template Functions section)
- Modify: `README.md:167-188` (Example Templates section)
- Modify: `README.md:7-8` (Warning block)

**Step 1: Update Template Functions section**

Replace lines 159-165 with:

```markdown
### Template Functions

**Color Formatting Functions:**
- `hex "path"` - hex with hash prefix (e.g., `#191724`)
- `bhex "path"` - bare hex without hash (e.g., `191724`)
- `hexa "path"` - hex with alpha channel (e.g., `#191724ff`)
- `bhexa "path"` - bare hex with alpha (e.g., `191724ff`)
- `rgb "path"` - RGB function format (e.g., `rgb(25, 23, 36)`)
- `rgba "path"` - RGBA with alpha (e.g., `rgba(25, 23, 36, 1.0)`)

All functions accept dot-notation palette paths (e.g., `"highlight.low"`, `"base"`).

**Style Access:**
- `style "path"` - returns a Style object with `.Bold`, `.Italic`, `.Underline` flags
```

**Step 2: Update Example Templates section**

Update the examples to use new syntax:

```markdown
**Ghostty terminal** (`ghostty.tmpl`):

```
background = {{ bhex .Theme.background }}
foreground = {{ bhex .Theme.foreground }}
cursor-color = {{ bhex .Theme.cursor }}
```

**Zed editor** (`zed.json.tmpl`):

```json
{
  "name": "{{ .Meta.Name }}",
  "style": {
    "background": "{{ hex .Theme.background }}",
    "editor.background": "{{ hex .Theme.background }}"
  }
}
```
```

**Step 3: Update warning block**

Change lines 7-8 to mention the breaking change:

```markdown
> [!WARNING]
> PaletteSwap is still in early development. The theme and templates formats are subject to breaking changes. Version 0.2.0 introduced a breaking change to the template API - see migration guide below.
```

**Step 4: Add migration guide section**

Add after the warning block (around line 9):

```markdown
### Migration from v0.1.x to v0.2.x

The template API has been redesigned for clarity. Update your custom templates:

**Old syntax (v0.1.x):**
```
{{ palette "highlight.low" | hex }}
{{ palette "base" | hexBare }}
```

**New syntax (v0.2.x):**
```
{{ hex "highlight.low" }}
{{ bhex "base" }}
```

**Conversion rules:**
- `palette "X" | hex` → `hex "X"`
- `palette "X" | hexBare` → `bhex "X"`
- `palette "X" | rgb` → `rgb "X"`
- Direct field access unchanged: `{{ hexBare .Theme.background }}` → `{{ bhex .Theme.background }}`
```

**Step 5: Commit**

```bash
git add README.md
git commit -m "docs: update README for new template API"
```

---

## Task 7: Run Full Test Suite and Manual Verification

**Files:**
- All modified files

**Step 1: Run complete test suite**

Run: `go test ./...`
Expected: All tests PASS

**Step 2: Build the binary**

Run: `go build -o paletteswap cmd/paletteswap/main.go`
Expected: Build succeeds

**Step 3: Generate themes**

Run: `./paletteswap generate`
Expected: All templates render successfully

**Step 4: Inspect generated output**

Run: `cat output/ghostty` and `cat output/zed.json`
Expected: Files contain properly formatted colors using new functions

**Step 5: Test error handling**

Create a temporary template with invalid path:
```bash
echo '{{ hex "nonexistent.path" }}' > /tmp/test.tmpl
```

Try to render it (will need to wire this up properly, or just verify the error handling code is in place).

**Step 6: Final commit if needed**

If any fixes were required:
```bash
git add .
git commit -m "fix: final adjustments for template API"
```

---

## Completion Checklist

- [ ] All new Color methods implemented and tested
- [ ] Template functions updated to new API
- [ ] Old palette function removed
- [ ] Example templates updated
- [ ] README documentation updated
- [ ] All tests passing
- [ ] Templates render successfully
- [ ] Error handling verified

## Testing Strategy

**Unit Tests:**
- Color methods (RGBA, HexAlpha, HexBareAlpha)
- Template function registration
- Path resolution with error cases

**Integration Tests:**
- Full template rendering
- Nested path access
- Error messages for invalid paths

**Manual Verification:**
- Generate actual theme files
- Inspect output formatting
- Verify migration guide accuracy
