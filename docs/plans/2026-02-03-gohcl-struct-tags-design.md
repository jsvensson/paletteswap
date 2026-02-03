# Design: Struct Tag-Based HCL Decoding

## Problem

The current `config.go` uses low-level `hclsyntax` APIs with manual iteration over blocks and attributes (~300 lines). This is verbose and hard to maintain as the schema evolves.

## Solution

Use `gohcl` struct tags for declarative decoding. Keep manual parsing only for the `syntax` block which has a mixed structure (flat attributes + nested style blocks) that can't be expressed with struct tags.

## Constraints

- Keep the current HCL schema exactly as-is
- Handle the two-pass requirement: `theme` and `ansi` blocks reference `palette` values via expressions like `palette.base`

## Design

### Struct Definitions

```go
// Meta holds theme metadata - no expressions, decodes directly
type Meta struct {
    Name       string `hcl:"name,attr"`
    Author     string `hcl:"author,attr"`
    Appearance string `hcl:"appearance,attr"`
}

// RawConfig captures palette first (no EvalContext needed)
type RawConfig struct {
    Palette map[string]string `hcl:"palette,block"`
    Remain  hcl.Body          `hcl:",remain"`
}

// ResolvedConfig decodes blocks that reference palette
type ResolvedConfig struct {
    Meta   *Meta             `hcl:"meta,block"`
    Theme  map[string]string `hcl:"theme,block"`
    ANSI   map[string]string `hcl:"ansi,block"`
    Syntax hcl.Body          `hcl:",remain"` // deferred for manual parsing
}
```

### Reusable Two-Pass Loader

```go
// Loader handles two-pass HCL decoding with palette resolution
type Loader struct {
    body    hcl.Body
    ctx     *hcl.EvalContext
    palette map[string]color.Color
}

// NewLoader parses HCL and builds the evaluation context
func NewLoader(path string) (*Loader, error) {
    src, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading theme file: %w", err)
    }

    file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
    if diags.HasErrors() {
        return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
    }

    // First pass: extract palette
    var raw RawConfig
    if diags := gohcl.DecodeBody(file.Body, nil, &raw); diags.HasErrors() {
        return nil, fmt.Errorf("decoding palette: %s", diags.Error())
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

// Decode decodes a value using the palette context.
// Reusable for any future blocks that reference palette.
func (l *Loader) Decode(target interface{}) error {
    if diags := gohcl.DecodeBody(l.body, l.ctx, target); diags.HasErrors() {
        return fmt.Errorf("decoding: %s", diags.Error())
    }
    return nil
}

// Palette returns the parsed palette colors
func (l *Loader) Palette() map[string]color.Color {
    return l.palette
}

// Context returns the EvalContext for manual parsing
func (l *Loader) Context() *hcl.EvalContext {
    return l.ctx
}
```

### Load Function

```go
// Load parses an HCL theme file and returns a fully-resolved Theme
func Load(path string) (*Theme, error) {
    loader, err := NewLoader(path)
    if err != nil {
        return nil, err
    }

    // Second pass: decode resolved blocks
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
    syntax, err := parseSyntax(resolved.Syntax, loader.Context())
    if err != nil {
        return nil, fmt.Errorf("parsing syntax: %w", err)
    }

    return &Theme{
        Meta:    *resolved.Meta,
        Palette: loader.Palette(),
        Theme:   themeColors,
        Syntax:  syntax,
        ANSI:    ansiColors,
    }, nil
}

// parseColorMap converts a string map to a color map
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

### Syntax Parsing (Kept Manual)

The `syntax` block has a mixed structure that gohcl can't express:
- Flat attributes: `keyword = palette.pine`
- Nested style blocks: `comment { color = palette.muted; italic = true }`

The existing `parseSyntaxBody`, `isStyleBlock`, and `parseStyleBlock` functions remain with minor adjustments to work with the new entry point:

```go
// parseSyntax handles the syntax block with its mixed structure
func parseSyntax(body hcl.Body, ctx *hcl.EvalContext) (color.ColorTree, error) {
    if body == nil {
        return make(color.ColorTree), nil
    }

    syntaxBody, ok := body.(*hclsyntax.Body)
    if !ok {
        return nil, fmt.Errorf("unexpected body type")
    }

    return parseSyntaxBody(syntaxBody, ctx)
}
```

## Changes Summary

### Removed (~150 lines)
- `parseMeta` - replaced by struct tags on `Meta`
- `parsePalette` - replaced by `map[string]string` decoding
- `parseColorBlock` - replaced by `map[string]string` decoding
- Manual block iteration in `Load`

### Kept (~120 lines)
- `parseSyntaxBody`, `isStyleBlock`, `parseStyleBlock` - syntax block complexity
- `buildEvalContext` - still needed for palette resolution

### Added (~50 lines)
- `Loader` type with `NewLoader`, `Decode`, `Palette`, `Context`
- `RawConfig`, `ResolvedConfig` struct definitions
- `parseColorMap` helper

### Net Result
- ~300 lines → ~170 lines
- Clearer separation: declarative decoding vs manual syntax parsing
- Reusable `Loader` for future two-pass decoding needs

## Testing

Existing tests should pass unchanged - the public API (`Load`) and return type (`Theme`) remain the same. Add unit tests for:
- `Loader` initialization and decoding
- `parseColorMap` error handling
