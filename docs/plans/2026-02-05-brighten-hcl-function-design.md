# HCL `brighten()` Function

## Overview

Add a `brighten(color, percentage)` function to the HCL theme format that calls the existing `color.Brighten()` function.

## Usage

```hcl
palette {
  base = "#191724"
  base_bright = brighten(base, 0.1)        # brighten palette color
  base_dark = brighten(base, -0.1)         # darken with negative value
}

theme {
  background = palette.base
  surface = brighten(palette.base, 0.05)   # derive inline
  highlight = brighten("#ffffff", -0.2)    # literal hex
}

syntax {
  comment {
    color = brighten(palette.muted, 0.1)
    italic = true
  }
}
```

## Function Signature

`brighten(color, percentage)` where:
- `color` - hex string or palette reference (resolves to hex string)
- `percentage` - float between -1.0 and 1.0 (negative values darken)

## Scope

The function is usable in all blocks: palette, theme, ansi, and syntax.

## Implementation

The change is localized to `internal/config/config.go`.

Register the function in `buildEvalContext()`:

```go
func buildEvalContext(palette color.ColorTree) *hcl.EvalContext {
    return &hcl.EvalContext{
        Variables: map[string]cty.Value{
            "palette": colorTreeToCty(palette),
        },
        Functions: map[string]function.Function{
            "brighten": makeBrightenFunc(),
        },
    }
}
```

The `makeBrightenFunc()` creates a go-cty function that:
1. Takes two parameters: string (color) and number (percentage)
2. Parses the hex string using `color.ParseHex()`
3. Calls `color.Brighten()` with the parsed color and percentage
4. Returns the result as a hex string

## Error Handling

Returns HCL diagnostics for:
- Invalid hex color format
- Wrong argument types

No clamping for percentage - values outside -1.0 to 1.0 hit `math.Min(1.0, ...)` in `Brighten()`.

## Files Changed

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `makeBrightenFunc()` and register in `buildEvalContext()` |
| `internal/config/config_test.go` | Add tests for brighten in various blocks |
