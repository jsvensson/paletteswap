# gohcl Struct Tags Refactor - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor config.go to use gohcl struct tags instead of manual HCL parsing, reducing code by ~150 lines while maintaining identical behavior.

**Architecture:** Two-pass decoding via a reusable `Loader` type. First pass extracts palette (literal values). Second pass decodes theme/ansi using an EvalContext built from palette. Syntax block remains manually parsed due to its mixed structure.

**Tech Stack:** `github.com/hashicorp/hcl/v2/gohcl`, existing `hclsyntax` for syntax block parsing.

**Worktree:** `.worktrees/gohcl-refactor` (branch: `feature/gohcl-struct-tags`)

---

## Task 1: Add gohcl import and struct definitions

**Files:**
- Modify: `internal/config/config.go:1-30`

**Step 1: Add gohcl import**

Add to imports:

```go
"github.com/hashicorp/hcl/v2/gohcl"
```

**Step 2: Add struct tag to Meta**

Replace the existing `Meta` struct:

```go
// Meta holds theme metadata.
type Meta struct {
	Name       string `hcl:"name,attr"`
	Author     string `hcl:"author,attr"`
	Appearance string `hcl:"appearance,attr"`
}
```

**Step 3: Add RawConfig and ResolvedConfig structs**

Add after the `Meta` struct:

```go
// RawConfig captures the palette block first (no EvalContext needed).
type RawConfig struct {
	Palette map[string]string `hcl:"palette,block"`
	Remain  hcl.Body          `hcl:",remain"`
}

// ResolvedConfig decodes blocks that reference palette.
type ResolvedConfig struct {
	Meta   *Meta             `hcl:"meta,block"`
	Theme  map[string]string `hcl:"theme,block"`
	ANSI   map[string]string `hcl:"ansi,block"`
	Remain hcl.Body          `hcl:",remain"` // captures syntax for manual parsing
}
```

**Step 4: Run tests to verify no regressions**

Run: `go test ./internal/config/...`
Expected: All tests pass (structs added but not yet used)

**Step 5: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add gohcl struct definitions

Add Meta struct tags and RawConfig/ResolvedConfig types
for two-pass HCL decoding. Not yet wired up."
```

---

## Task 2: Add Loader type

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add Loader struct and NewLoader function**

Add after the struct definitions:

```go
// Loader handles two-pass HCL decoding with palette resolution.
type Loader struct {
	body    hcl.Body
	ctx     *hcl.EvalContext
	palette map[string]color.Color
}

// NewLoader parses an HCL file and builds the evaluation context from palette.
func NewLoader(path string) (*Loader, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading theme file: %w", err)
	}

	file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	// First pass: extract palette (literal values, no context needed)
	var raw RawConfig
	if diags := gohcl.DecodeBody(file.Body, nil, &raw); diags.HasErrors() {
		return nil, fmt.Errorf("decoding palette: %s", diags.Error())
	}

	if len(raw.Palette) == 0 {
		return nil, fmt.Errorf("no palette block found")
	}

	palette, err := parseColorMap(raw.Palette)
	if err != nil {
		return nil, fmt.Errorf("parsing palette: %w", err)
	}

	return &Loader{
		body:    file.Body,
		ctx:     buildEvalContext(palette),
		palette: palette,
	}, nil
}
```

**Step 2: Add Loader methods**

```go
// Decode decodes a value using the palette context.
// Reusable for any blocks that reference palette values.
func (l *Loader) Decode(target interface{}) error {
	if diags := gohcl.DecodeBody(l.body, l.ctx, target); diags.HasErrors() {
		return fmt.Errorf("decoding: %s", diags.Error())
	}
	return nil
}

// Palette returns the parsed palette colors.
func (l *Loader) Palette() map[string]color.Color {
	return l.palette
}

// Context returns the EvalContext for manual parsing.
func (l *Loader) Context() *hcl.EvalContext {
	return l.ctx
}
```

**Step 3: Add parseColorMap helper**

```go
// parseColorMap converts a map of hex strings to a map of Colors.
func parseColorMap(m map[string]string) (map[string]color.Color, error) {
	result := make(map[string]color.Color, len(m))
	for name, hex := range m {
		c, err := color.ParseHex(hex)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		result[name] = c
	}
	return result, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/...`
Expected: All tests pass (Loader added but not yet used)

**Step 5: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add Loader type for two-pass decoding

Loader handles parsing, palette extraction, and EvalContext
building. Provides Decode() method for reusable decoding."
```

---

## Task 3: Add parseSyntax entry point

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add parseSyntax function**

This bridges the new Loader approach to the existing syntax parsing. Add before `parseSyntaxBody`:

```go
// parseSyntax extracts and parses the syntax block from an hcl.Body.
// It handles the mixed structure (flat attributes + nested style blocks).
func parseSyntax(body hcl.Body, ctx *hcl.EvalContext) (color.ColorTree, error) {
	if body == nil {
		return make(color.ColorTree), nil
	}

	// The remain body contains unparsed blocks including syntax.
	// We need to find the syntax block within it.
	syntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// If not hclsyntax.Body, return empty tree (no syntax block)
		return make(color.ColorTree), nil
	}

	// Find the syntax block
	for _, block := range syntaxBody.Blocks {
		if block.Type == "syntax" {
			return parseSyntaxBody(block.Body, ctx)
		}
	}

	return make(color.ColorTree), nil
}
```

**Step 2: Run tests**

Run: `go test ./internal/config/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add parseSyntax entry point

Bridge function to extract syntax block from remain body
and delegate to existing parseSyntaxBody."
```

---

## Task 4: Rewrite Load() to use Loader

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Replace Load function**

Replace the entire `Load` function with:

```go
// Load parses an HCL theme file and returns a fully-resolved Theme.
func Load(path string) (*Theme, error) {
	loader, err := NewLoader(path)
	if err != nil {
		return nil, err
	}

	// Second pass: decode blocks that reference palette
	var resolved ResolvedConfig
	if err := loader.Decode(&resolved); err != nil {
		return nil, err
	}

	// Convert string maps to color maps
	themeColors, err := parseColorMap(resolved.Theme)
	if err != nil {
		return nil, fmt.Errorf("parsing theme: %w", err)
	}

	ansiColors, err := parseColorMap(resolved.ANSI)
	if err != nil {
		return nil, fmt.Errorf("parsing ansi: %w", err)
	}

	// Parse syntax manually (nested blocks with style properties)
	syntax, err := parseSyntax(resolved.Remain, loader.Context())
	if err != nil {
		return nil, fmt.Errorf("parsing syntax: %w", err)
	}

	meta := Meta{}
	if resolved.Meta != nil {
		meta = *resolved.Meta
	}

	return &Theme{
		Meta:    meta,
		Palette: loader.Palette(),
		Theme:   themeColors,
		Syntax:  syntax,
		ANSI:    ansiColors,
	}, nil
}
```

**Step 2: Run tests**

Run: `go test ./internal/config/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): rewrite Load() to use Loader

Load now uses two-pass gohcl decoding via Loader.
All existing tests should pass unchanged."
```

---

## Task 5: Remove obsolete parsing functions

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Delete parseMeta function**

Remove the entire `parseMeta` function (~25 lines).

**Step 2: Delete parsePalette function**

Remove the entire `parsePalette` function (~25 lines).

**Step 3: Delete parseColorBlock function**

Remove the entire `parseColorBlock` function (~25 lines).

**Step 4: Clean up unused imports**

Remove `"sort"` from imports if no longer used (check if buildEvalContext still uses it).

**Step 5: Run tests**

Run: `go test ./internal/config/...`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor(config): remove obsolete parsing functions

Delete parseMeta, parsePalette, parseColorBlock - now handled
by gohcl struct tag decoding."
```

---

## Task 6: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All packages pass

**Step 2: Run with actual theme file**

Run: `go run ./cmd/paletteswap -t ../../theme.hcl -o ghostty`
Expected: Produces valid Ghostty config output

**Step 3: Compare output**

Run from main worktree and compare outputs are identical:
```bash
cd ../.. && go run ./cmd/paletteswap -t theme.hcl -o ghostty > /tmp/old.txt
cd .worktrees/gohcl-refactor && go run ./cmd/paletteswap -t ../../theme.hcl -o ghostty > /tmp/new.txt
diff /tmp/old.txt /tmp/new.txt
```
Expected: No differences

**Step 4: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore(config): final cleanup after gohcl refactor"
```

---

## Summary

| Task | Description | Est. Lines Changed |
|------|-------------|-------------------|
| 1 | Add struct definitions | +25 |
| 2 | Add Loader type | +60 |
| 3 | Add parseSyntax entry point | +20 |
| 4 | Rewrite Load() | +35, -45 |
| 5 | Remove obsolete functions | -75 |
| 6 | Verify | 0 |

**Net change:** ~-40 lines, much cleaner separation of concerns.
