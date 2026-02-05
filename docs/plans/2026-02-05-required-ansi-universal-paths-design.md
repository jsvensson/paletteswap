# Required ANSI Block + Universal Template Path API

**Date:** 2026-02-05
**Status:** Design
**Breaking Change:** Yes

## Summary

Make the ANSI block required with validation for all 16 standard terminal colors, and extend the template API redesign to support universal dot-notation paths across all blocks (palette, theme, ansi, syntax), not just palette.

## Overview & Goals

### Problem Statement

1. **ANSI completeness:** Terminal color themes should always define all 16 standard ANSI colors, but there's currently no validation to ensure this
2. **Inconsistent template access:** The proposed template API redesign only supports dot-notation for palette colors, requiring mixed syntax for different blocks

### Proposed Solution

1. **Required ANSI validation:** Validate during HCL parsing that all 16 ANSI colors are present
2. **Universal path API:** Extend `hex`, `bhex`, etc. to accept paths from any block: `"palette.base"`, `"theme.background"`, `"ansi.black"`, `"syntax.keyword"`

### Key Decisions

- **ANSI remains separate:** Don't merge ANSI into syntax or theme blocks
- **All 16 colors required:** No optional colors, no extras allowed
- **Universal path syntax:** One consistent API for accessing colors across all blocks
- **Backward compatibility (temporary):** Keep supporting direct field access like `{{ hex .Theme.background }}` for gradual migration
- **Complete API rename:** Replace `hexBare` with `bhex` as part of this change

## Part 1: Required ANSI Block Validation

### Required Color Names

All 16 standard terminal colors must be present:

```hcl
ansi {
  # Base 8 colors
  black   = ...
  red     = ...
  green   = ...
  yellow  = ...
  blue    = ...
  magenta = ...
  cyan    = ...
  white   = ...

  # Bright variants
  bright_black   = ...
  bright_red     = ...
  bright_green   = ...
  bright_yellow  = ...
  bright_blue    = ...
  bright_magenta = ...
  bright_cyan    = ...
  bright_white   = ...
}
```

### Validation Logic

**When:** During HCL parsing/config loading, before any template rendering

**What to check:**
1. ANSI block exists
2. All 16 color names are present (exact match required)
3. No missing colors
4. No extra/unknown color names (optional strictness)

**Error format:**
```
Error: ansi block incomplete
Missing colors: bright_red, bright_blue, bright_cyan
Required colors: black, red, green, yellow, blue, magenta, cyan, white,
                 bright_black, bright_red, bright_green, bright_yellow,
                 bright_blue, bright_magenta, bright_cyan, bright_white
```

### Implementation Notes

- Define required color list as a constant/slice
- Validate after HCL parsing, before template execution
- Exit with non-zero status code on validation failure
- Consider whether to also validate that each color has a valid value (non-empty)

## Part 2: Universal Template Path API

### Current State (Template API Redesign)

The template API redesign proposes:
```
{{ palette "highlight.low" | hex }}  →  {{ hex "highlight.low" }}
```

But this only works for palette paths, requiring continued use of field access for other blocks:
```
{{ hex .Theme.background }}  # Still needed for theme
{{ hex .ANSI.black }}        # Still needed for ansi
```

### Universal Path Extension

Extend the new `hex`, `bhex`, etc. functions to accept paths from ALL blocks:

```go
// Palette colors
{{ hex "palette.base" }}
{{ hex "palette.highlight.low" }}

// Theme colors
{{ hex "theme.background" }}
{{ hex "theme.foreground" }}

// ANSI colors
{{ hex "ansi.black" }}
{{ hex "ansi.bright_red" }}

// Syntax colors
{{ hex "syntax.keyword" }}
{{ hex "syntax.comment" }}
```

### Path Resolution Logic

**Format:** `"block.path.to.color"`

**Algorithm:**
1. Parse dot-notation path into segments
2. Identify block from first segment: "palette", "theme", "ansi", "syntax"
3. Route to appropriate data structure (`.Palette`, `.Theme`, `.ANSI`, `.Syntax`)
4. Navigate remaining path segments for nested access (e.g., "palette.highlight.low" → navigate to highlight → low)
5. Extract color value
6. Format using function type (hex, bhex, rgb, etc.)
7. Return error if path doesn't exist

**Example path resolution:**
- `"palette.highlight.low"` → `.Palette` → navigate("highlight") → navigate("low") → extract color
- `"theme.background"` → `.Theme` → field("background") → extract color
- `"ansi.black"` → `.ANSI` → field("black") → extract color
- `"syntax.keyword"` → `.Syntax` → field("keyword") → extract color only (ignore style flags)

### Function Set

All six functions from the template API redesign support universal paths:

| Function | Example Output | Description |
|----------|---------------|-------------|
| `hex` | `#112233` | Hex with hash prefix |
| `bhex` | `112233` | Bare hex (no hash) |
| `hexa` | `#112233ff` | Hex with alpha (ff = opaque) |
| `bhexa` | `112233ff` | Bare hex with alpha |
| `rgb` | `rgb(17, 34, 51)` | RGB function format |
| `rgba` | `rgba(17, 34, 51, 1.0)` | RGBA with alpha |

### Backward Compatibility (Temporary)

Keep supporting direct field access during migration period:

```go
{{ hex .Theme.background }}  # Old way - still works
{{ hex "theme.background" }} # New way - preferred
```

This allows gradual template migration. Can be deprecated in a future version.

### Syntax Block Special Handling

The `syntax` block contains both colors AND style flags. When accessing via universal paths, only return the color:

```go
// Color access
{{ hex "syntax.keyword" }}  # Returns color only

// Style flags (unchanged)
{{ if (style "syntax.keyword").Bold }}  # Returns style object
```

The existing `style` function remains unchanged. Future improvements to style flag access are out of scope for this design.

## Part 3: Template Updates

### Zed Template Conversion

**Current patterns:**
```go
"background": "{{ hex .Theme.background }}"
"terminal.ansi.black": "{{ hex .ANSI.black }}"
{{- with .Syntax.keyword }}
"keyword": {
  "color": "{{ .Color | hex }}"
}
{{- end }}
```

**After conversion:**
```go
"background": "{{ hex "theme.background" }}"
"terminal.ansi.black": "{{ hex "ansi.black" }}"
{{- with .Syntax.keyword }}
"keyword": {
  "color": "{{ hex "syntax.keyword" }}"
}
{{- end }}
```

**Notes:**
- `with` blocks remain necessary for checking existence and accessing style flags
- All direct field access converted to universal paths
- All `hexBare` converted to `bhex`

### Other Templates

Any example templates in docs or templates directory should be updated to use the new API.

## Implementation Approach

### Phase 1: Template API Implementation

1. **Add universal path resolution:**
   - Create path parser (split on dots, identify block)
   - Create block router (switch on first segment)
   - Create nested path navigator (for palette paths like "highlight.low")
   - Handle color extraction from each block type

2. **Register new functions:**
   - `hex`, `bhex`, `hexa`, `bhexa`, `rgb`, `rgba` with universal path support
   - Each function takes `interface{}` to accept both strings (new) and direct field access (backward compat)
   - Type switch to handle both patterns

3. **Remove old functions:**
   - Remove `hexBare` (replaced by `bhex`)
   - Remove old `palette` function with piping syntax
   - Remove old filter-style hex/rgb formatters

### Phase 2: ANSI Validation

1. **Define required colors:**
   ```go
   var requiredANSIColors = []string{
       "black", "red", "green", "yellow",
       "blue", "magenta", "cyan", "white",
       "bright_black", "bright_red", "bright_green", "bright_yellow",
       "bright_blue", "bright_magenta", "bright_cyan", "bright_white",
   }
   ```

2. **Add validation function:**
   ```go
   func validateANSI(ansi ANSIBlock) error {
       // Check each required color is present
       // Return error listing missing colors
   }
   ```

3. **Call during config loading:**
   - After HCL parsing completes
   - Before template rendering starts
   - Exit with error if validation fails

### Phase 3: Template Updates

1. Update `templates/zed.json.tmpl`
2. Update any other template files
3. Update README examples

### Phase 4: Documentation

1. Update README template function section
2. Add migration guide for breaking changes
3. Update example code blocks
4. Add note about backward compatibility window

## Testing Strategy

### Template API Tests

**Path resolution:**
- Test each block type: palette, theme, ansi, syntax
- Test nested paths: "palette.highlight.low"
- Test single-level paths: "theme.background"
- Test invalid paths (should error)
- Test invalid block names (should error)

**Function output:**
- Test each function (hex, bhex, hexa, bhexa, rgb, rgba)
- Verify output format for each
- Test alpha defaults (ff / 1.0)

**Backward compatibility:**
- Test direct field access still works: `{{ hex .Theme.background }}`
- Test new path syntax: `{{ hex "theme.background" }}`

### ANSI Validation Tests

**Valid cases:**
- Complete ANSI block with all 16 colors
- Different palette color assignments

**Invalid cases:**
- Missing ANSI block entirely
- Missing one color
- Missing multiple colors
- Extra/unknown color names

**Error messages:**
- Verify error lists missing colors
- Verify error shows required color list

### Integration Tests

- Generate complete themes with new template API
- Verify Zed template output is valid JSON
- Verify generated theme files work in target applications

## Migration Impact

### Breaking Changes

**Templates break:**
- Old `hexBare` function → must use `bhex`
- Old `palette "path" | hex` syntax → must use `hex "palette.path"`

**Themes break:**
- Missing ANSI colors → must add all 16 colors

### Migration Path

**For templates:**
1. Replace `hexBare` with `bhex`
2. Replace `palette "X" | hex` with `hex "palette.X"`
3. Optionally convert field access to universal paths

**For themes:**
1. Ensure ANSI block exists
2. Add any missing colors from the required 16

### What Doesn't Break

- HCL theme format (except ANSI validation)
- Meta block access
- Direct field access (backward compat maintained)
- Style function

## Future Work

- Configurable alpha channel values
- Improved style flag access API (ideas pending)
- Deprecate backward-compatible field access
- Additional color format functions as needed
- Consider strict validation for other blocks (theme, syntax)
