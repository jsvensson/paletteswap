# Package Restructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename `color.ColorTree` to `color.Tree`, rename `internal/config` to `internal/parser`, move `Theme`/`Meta`/`Load` to root package, and collapse `internal/engine` into root package.

**Architecture:** Four sequential refactoring steps. Each step produces a compilable, test-passing codebase. The rename steps use mechanical find-and-replace. The move steps require creating new files and updating imports.

**Tech Stack:** Go 1.25, HCL v2, cobra, go-cty

---

### Task 1: Rename `color.ColorTree` to `color.Tree`

**Files:**
- Modify: `internal/color/color.go:24` (type definition)
- Modify: `internal/config/config.go` (all references)
- Modify: `internal/config/config_test.go` (all references)
- Modify: `internal/engine/engine.go` (all references)
- Modify: `internal/engine/engine_test.go` (all references)
- Modify: `internal/engine/functions_test.go` (all references)
- Modify: `internal/engine/path_test.go` (all references)

**Step 1: Rename the type definition**

In `internal/color/color.go`, change:
```go
// ColorTree represents a nested map of colors, used for syntax scopes.
// Values are either Style or ColorTree.
type ColorTree map[string]any
```
to:
```go
// Tree represents a nested map of colors, used for syntax scopes.
// Values are either Style or Tree.
type Tree map[string]any
```

**Step 2: Replace all `color.ColorTree` references**

In every `.go` file under `internal/`, replace `color.ColorTree` with `color.Tree` and `ColorTree` (when used without qualifier inside `color` package) with `Tree`.

Files and approximate occurrence counts:
- `internal/config/config.go`: ~15 occurrences of `color.ColorTree`
- `internal/config/config_test.go`: ~4 occurrences of `color.ColorTree`
- `internal/engine/engine.go`: ~4 occurrences of `color.ColorTree`
- `internal/engine/engine_test.go`: ~6 occurrences of `color.ColorTree`
- `internal/engine/functions_test.go`: ~3 occurrences of `color.ColorTree`
- `internal/engine/path_test.go`: ~3 occurrences of `color.ColorTree`

**Step 3: Run tests**

Run: `cd /Users/echo/git/github.com/jsvensson/paletteswap && go test ./...`
Expected: All tests pass.

**Step 4: Commit**

```bash
git add internal/color/color.go internal/config/config.go internal/config/config_test.go internal/engine/engine.go internal/engine/engine_test.go internal/engine/functions_test.go internal/engine/path_test.go
git commit -m "refactor: rename color.ColorTree to color.Tree"
```

---

### Task 2: Rename `internal/config` to `internal/parser`

**Files:**
- Rename: `internal/config/` -> `internal/parser/`
- Modify: `internal/parser/config.go` (package declaration)
- Modify: `internal/parser/config_test.go` (package declaration)
- Modify: `internal/parser/ansi_test.go` (package declaration)
- Modify: `internal/engine/engine.go` (import path)
- Modify: `internal/engine/engine_test.go` (import path + all `config.` references become `parser.`)
- Modify: `internal/engine/functions_test.go` (import path + all `config.` references become `parser.`)
- Modify: `cmd/paletteswap/main.go` (import path)

**Step 1: Rename the directory**

```bash
cd /Users/echo/git/github.com/jsvensson/paletteswap
git mv internal/config internal/parser
```

**Step 2: Update package declarations**

In all `.go` files under `internal/parser/`, change `package config` to `package parser`.

Files:
- `internal/parser/config.go`
- `internal/parser/config_test.go`
- `internal/parser/ansi_test.go`

**Step 3: Update import paths and qualifier references**

In `internal/engine/engine.go`:
- Change import `"github.com/jsvensson/paletteswap/internal/config"` to `"github.com/jsvensson/paletteswap/internal/parser"`
- Change all `config.Theme` to `parser.Theme`
- Change all `config.Meta` to `parser.Meta`

In `internal/engine/engine_test.go`:
- Change import `"github.com/jsvensson/paletteswap/internal/config"` to `"github.com/jsvensson/paletteswap/internal/parser"`
- Change all `config.Theme` to `parser.Theme`
- Change all `config.Meta` to `parser.Meta`

In `internal/engine/functions_test.go`:
- Change import `"github.com/jsvensson/paletteswap/internal/config"` to `"github.com/jsvensson/paletteswap/internal/parser"`
- Change all `config.Theme` to `parser.Theme`

In `cmd/paletteswap/main.go`:
- Change import `"github.com/jsvensson/paletteswap/internal/config"` to `"github.com/jsvensson/paletteswap/internal/parser"`
- Change `config.Load` to `parser.Load`

**Step 4: Run tests**

Run: `cd /Users/echo/git/github.com/jsvensson/paletteswap && go test ./...`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: rename internal/config to internal/parser"
```

---

### Task 3: Move `Theme`, `Meta`, `Load()` to root package

This creates the root `paletteswap` package with `theme.go`. The `Load` function delegates to `internal/parser` for HCL parsing, then assembles the result.

**Files:**
- Create: `theme.go` (root package — `package paletteswap`)
- Modify: `internal/parser/config.go` (remove `Theme`/`Meta` structs, export a `Parse` function returning raw pieces)
- Modify: `internal/engine/engine.go` (update imports — `parser.Theme` becomes root package `Theme`, but we defer this to Task 4 since engine moves to root too)

Since Task 4 moves engine into the root package, we handle both moves together to avoid a transient circular dependency. The approach:

**Step 1: Create `theme.go` in root package**

Create `theme.go`:
```go
package paletteswap

import (
	"fmt"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/parser"
)

// Theme is the fully-resolved theme data, ready for template rendering.
type Theme struct {
	Meta    Meta
	Palette color.Tree
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}

// Meta holds theme metadata.
type Meta struct {
	Name       string
	Author     string
	Appearance string
	URL        string
}

// Load parses an HCL theme file and returns a fully-resolved Theme.
func Load(path string) (*Theme, error) {
	raw, err := parser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("loading theme: %w", err)
	}

	return &Theme{
		Meta: Meta{
			Name:       raw.Meta.Name,
			Author:     raw.Meta.Author,
			Appearance: raw.Meta.Appearance,
			URL:        raw.Meta.URL,
		},
		Palette: raw.Palette,
		Theme:   raw.Theme,
		Syntax:  raw.Syntax,
		ANSI:    raw.ANSI,
	}, nil
}
```

**Step 2: Add `Parse` function to `internal/parser/config.go`**

The existing `Load` function becomes `Parse`, and its return type changes to `ParseResult` (a new internal struct with the same fields as the old `Theme`). The `Meta` struct keeps its HCL tags (needed for parsing) and stays in `parser`.

Add to `internal/parser/config.go`:
```go
// ParseResult holds the raw parsed theme data.
type ParseResult struct {
	Meta    Meta
	Palette color.Tree
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}

// Parse parses an HCL theme file and returns the raw parsed data.
func Parse(path string) (*ParseResult, error) {
	// ... same body as current Load(), but returns ParseResult
}
```

Then rename the existing `Load` to `Parse`, change return type from `*Theme` to `*ParseResult`, and update the return statement to use `ParseResult` field names.

Remove the old `Theme` struct from parser (it's now in the root package). Keep `Meta` in parser since it has HCL struct tags needed for decoding.

**Step 3: This step is deferred — engine import updates happen in Task 4.**

Do NOT run tests yet — engine still references `parser.Theme` which no longer exists. Proceed directly to Task 4.

---

### Task 4: Move `internal/engine` into root package

**Files:**
- Create: `engine.go` (move from `internal/engine/engine.go`, change package to `paletteswap`)
- Create: `engine_test.go` (move from `internal/engine/engine_test.go`)
- Create: `functions_test.go` (move from `internal/engine/functions_test.go`)
- Create: `path_test.go` (move from `internal/engine/path_test.go`)
- Delete: `internal/engine/` directory
- Modify: `cmd/paletteswap/main.go` (update imports)

**Step 1: Move engine code to root package**

Move files and update package declarations:
```bash
cd /Users/echo/git/github.com/jsvensson/paletteswap
git mv internal/engine/engine.go engine.go
git mv internal/engine/engine_test.go engine_test.go
git mv internal/engine/functions_test.go functions_test.go
git mv internal/engine/path_test.go path_test.go
```

**Step 2: Update `engine.go`**

- Change `package engine` to `package paletteswap`
- Remove import of `"github.com/jsvensson/paletteswap/internal/config"` (no longer needed)
- Remove import of `"github.com/jsvensson/paletteswap/internal/engine"` if present
- Update `templateData` to use `Meta` directly (not `config.Meta` or `parser.Meta`)
- Update `func (e *Engine) Run(theme *config.Theme)` to `func (e *Engine) Run(theme *Theme)`
- Update `func buildTemplateData(theme *config.Theme)` to `func buildTemplateData(theme *Theme)`

**Step 3: Update test files**

In `engine_test.go`, `functions_test.go`, `path_test.go`:
- Change `package engine` to `package paletteswap`
- Remove import of `"github.com/jsvensson/paletteswap/internal/config"`
- Change all `config.Theme` to `Theme`
- Change all `config.Meta` to `Meta`
- Change all `color.ColorTree` to `color.Tree` (should already be done from Task 1)

**Step 4: Update `cmd/paletteswap/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/jsvensson/paletteswap"
	"github.com/spf13/cobra"
)
```

Change:
- `config.Load(flagTheme)` to `paletteswap.Load(flagTheme)`
- `engine.Engine{...}` to `paletteswap.Engine{...}`
- Remove old `internal/config` and `internal/engine` imports

**Step 5: Remove empty `internal/engine` directory**

```bash
rmdir internal/engine
```

If the directory isn't empty (e.g., leftover files), investigate before removing.

**Step 6: Run tests**

Run: `cd /Users/echo/git/github.com/jsvensson/paletteswap && go test ./...`
Expected: All tests pass.

**Step 7: Commit**

```bash
git add -A
git commit -m "refactor: move Theme, Meta, Load, Engine to root package"
```

---

### Task 5: Clean up

**Step 1: Verify the final structure**

Expected file layout:
```
paletteswap/
  theme.go          # Theme, Meta, Load()
  engine.go         # Engine, Run(), template funcs
  engine_test.go
  functions_test.go
  path_test.go
  cmd/paletteswap/main.go
  internal/
    color/
      color.go      # Color, Style, Tree
      color_test.go
      functions.go   # Brighten, Darken
    parser/
      config.go     # ParseResult, Meta (with HCL tags), Parse(), Loader, etc.
      config_test.go
      ansi_test.go
```

**Step 2: Run full test suite one more time**

Run: `cd /Users/echo/git/github.com/jsvensson/paletteswap && go test ./... -v`
Expected: All tests pass.

**Step 3: Run `go vet`**

Run: `cd /Users/echo/git/github.com/jsvensson/paletteswap && go vet ./...`
Expected: No issues.
