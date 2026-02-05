# Template API Redesign

**Date:** 2026-02-05
**Status:** Design
**Breaking Change:** Yes

## Summary

Redesign the template API to replace the vague `palette` function with piping syntax with direct, precisely-named functions. This makes the template API clearer and more intuitive while adding support for alpha channel output (with hardcoded defaults for now).

## Overview & Goals

### Current Problem

The `palette` function with piping syntax is vague and indirect:
```
{{ palette "highlight.low" | hex }}
{{ palette "base" | hexBare }}
```

### Proposed Solution

Direct, precisely-named functions that take palette paths:
```
{{ hex "highlight.low" }}
{{ bhex "base" }}
```

### Key Decisions

- **Breaking change**: Remove `palette` function entirely (acceptable in early development)
- **Dot-notation only**: Functions only accept palette paths like `"highlight.low"`, not literal hex values
- **Alpha defaults**: New alpha variants (`hexa`, `bhexa`, `rgba`) default to full opacity (`ff` / `1.0`)
- **Keep `style` unchanged**: Style flag access stays as is for now

### New Function Set

| Function | Example Output | Description |
|----------|---------------|-------------|
| `hex` | `#112233` | Hex with hash prefix |
| `bhex` | `112233` | Bare hex (no hash) |
| `hexa` | `#112233ff` | Hex with alpha (ff = opaque) |
| `bhexa` | `112233ff` | Bare hex with alpha |
| `rgb` | `rgb(17, 34, 51)` | RGB function format |
| `rgba` | `rgba(17, 34, 51, 1.0)` | RGBA with alpha as float |

## Migration Impact

### What Breaks

All existing templates using the `palette` function with piping will break:
```
# Old syntax (will not work)
{{ palette "highlight.low" | hex }}
{{ palette "base" | hexBare }}
{{ palette "text" | rgb }}
```

### Migration Path

Simple find-replace pattern:
```
# New syntax
{{ hex "highlight.low" }}
{{ bhex "base" }}
{{ rgb "text" }}
```

### Conversion Rules

- `palette "X" | hex` → `hex "X"`
- `palette "X" | hexBare` → `bhex "X"`
- `palette "X" | rgb` → `rgb "X"`

### What Doesn't Break

- The `style` function stays unchanged
- HCL theme format unchanged
- Direct field access like `.Theme.background` unchanged
- All other template functions work as before

### Example Template Before/After

Before:
```
background = {{ hexBare .Theme.background }}
foreground = {{ hexBare .Theme.foreground }}
color1 = {{ palette "highlight.low" | hex }}
```

After:
```
background = {{ bhex .Theme.background }}
foreground = {{ bhex .Theme.foreground }}
color1 = {{ hex "highlight.low" }}
```

## Implementation Approach

### Template Function Registration

All six new functions (`hex`, `bhex`, `hexa`, `bhexa`, `rgb`, `rgba`) will be registered in the Go template engine's function map. Each function:
- Takes a string argument (dot-notation palette path)
- Resolves the path through the nested palette structure
- Returns the color formatted as a string

### Color Resolution

Functions need to navigate the nested palette structure, similar to how the existing `palette` function works:
- Parse dot-notation path (e.g., `"highlight.low"` → `["highlight", "low"]`)
- Traverse the palette tree to find the color
- Return error if path doesn't exist

### Alpha Channel Handling

For this iteration, alpha is hardcoded:
- Hex functions (`hexa`, `bhexa`): append `"ff"`
- RGBA function: append `, 1.0` to the RGB values
- Configurable alpha comes in a future feature

### Error Handling

If a palette path doesn't exist, the template rendering should fail with a clear error message indicating which path was not found.

### Code Changes

- Remove old `palette` function and piping formatters (`hex`, `hexBare`, `rgb` as filters)
- Add new standalone functions with path resolution
- Update function map registration
- Keep existing color formatting logic (hex conversion, RGB conversion)

## Documentation Updates

### README Sections to Update

#### 1. Template Functions Section (currently lines 159-165)

Replace the current list with:

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

#### 2. Example Templates Section (lines 167-188)

Update the Ghostty and Zed examples to use new syntax:

```
# Ghostty - before
background = {{ hexBare .Theme.background }}

# Ghostty - after
background = {{ bhex .Theme.background }}
```

#### 3. Add Migration Note

Add to the warning block at the top mentioning this is a breaking change from previous versions.

## Future Work

- Configurable alpha channel values (in theme HCL or as function parameters)
- Additional color format functions as needed
- Potential redesign of `style` function syntax
