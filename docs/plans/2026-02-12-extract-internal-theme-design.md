# Extract Shared HCL Helpers into `internal/theme`

## Goal

Eliminate ~170 lines of duplicated code between `internal/parser/config.go` and `internal/lsp/analyzer.go` by extracting shared HCL evaluation helpers into a new `internal/theme` package.

Both packages independently implement identical functions for converting palette nodes to HCL evaluation contexts. The duplication was introduced to avoid coupling the LSP to the parser, but a shared utility package achieves the same decoupling while removing the maintenance burden of keeping two copies in sync.

## New Package: `internal/theme`

A single file `theme.go` containing these exported symbols, moved verbatim from `parser/config.go`:

| Symbol | Type | Purpose |
|--------|------|---------|
| `RequiredANSIColors` | `[]string` | The 16 standard terminal color names |
| `ResolveColor()` | `func(cty.Value) (string, error)` | Extract color hex from a cty.Value |
| `NodeToCty()` | `func(*color.Node) cty.Value` | Convert color.Node tree to cty.Value |
| `MakeBrightenFunc()` | `func() function.Function` | HCL function factory for `brighten()` |
| `MakeDarkenFunc()` | `func() function.Function` | HCL function factory for `darken()` |
| `BuildEvalContext()` | `func(*color.Node) *hcl.EvalContext` | Build HCL eval context with palette vars and functions |

**Dependencies:** `internal/color`, `go-cty`, `hashicorp/hcl/v2` only.

## Changes to `internal/parser/config.go`

- Remove the 6 functions/vars listed above (~120 lines)
- Add import `"github.com/jsvensson/paletteswap/internal/theme"`
- Replace call sites: `resolveColor(...)` becomes `theme.ResolveColor(...)`, etc.
- Remove `"github.com/zclconf/go-cty/cty/function"` import (only used by the moved function factories)
- Remove `requiredANSIColors` var and its references become `theme.RequiredANSIColors`
- `paletteItem` struct stays (not duplicated)

## Changes to `internal/lsp/analyzer.go`

- Remove the 6 duplicated functions/vars (~120 lines)
- Add import `"github.com/jsvensson/paletteswap/internal/theme"`
- Replace call sites: `analyzerResolveColor(...)` becomes `theme.ResolveColor(...)`, etc.
- Remove `"github.com/zclconf/go-cty/cty/function"` import
- `blockItem` struct stays (not duplicated)

## Changes to `internal/lsp/completion.go`

- Replace `requiredANSIColors` reference with `theme.RequiredANSIColors`
- Add import `"github.com/jsvensson/paletteswap/internal/theme"`

## What doesn't change

- No behavior changes; functions are moved verbatim
- Existing tests in `parser/config_test.go` stay, calling `theme.ResolveColor` instead of `resolveColor` (tests for exported functions move to `internal/theme/`)
- `theme.go` in root package is unaffected (public API: `paletteswap.Theme`, `paletteswap.Load()`)

## Dependency graph

```
internal/color  (no HCL deps)
    ^
    |
internal/theme  (color + HCL eval helpers)
    ^       ^
    |       |
parser     lsp  (no dependency between them)
```

## Net effect

- `parser/config.go`: ~590 -> ~470 lines
- `lsp/analyzer.go`: ~784 -> ~664 lines
- New `internal/theme/theme.go`: ~130 lines
- Total lines removed: ~110 (net reduction from deduplication)
