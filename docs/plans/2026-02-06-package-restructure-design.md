# Package Restructure Design

## Goal

Reduce stuttering and improve the public API surface by restructuring packages.

## Changes

### 1. Rename `color.ColorTree` to `color.Tree`

Straightforward rename across the codebase. Every reference to `color.ColorTree` becomes `color.Tree`. No backwards-compat shim.

### 2. Rename `internal/config` to `internal/parser`

Same contents (all HCL parsing machinery), more accurate name. Contains `Loader`, `RawConfig`, `ResolvedConfig`, `PaletteBlock`, `ColorBlock`, `buildEvalContext`, `parsePaletteBody`, etc.

### 3. Move `Theme`, `Meta`, `Load()` to root `paletteswap` package

New file `theme.go` in the root package containing:

- `Theme` struct — same fields, references `color.Tree` and `color.Color`
- `Meta` struct — same fields with HCL struct tags
- `Load(path string) (*Theme, error)` — public entry point, delegates HCL parsing to `internal/parser`

### 4. Move `internal/engine` into root `paletteswap` package

The template engine code moves to `engine.go` in the root package. This avoids a circular dependency (internal/engine would otherwise need to import the root package for `Theme`).

Root package files after restructure:
- `theme.go` — `Theme`, `Meta`, `Load()`
- `engine.go` — `Engine`, `Run()`, template data building, template functions

### Result

- `cmd/paletteswap/main.go` uses `paletteswap.Load()` and `paletteswap.Engine{}`
- `internal/parser` keeps all HCL parsing internals
- `internal/color` keeps color types with `Tree` instead of `ColorTree`
