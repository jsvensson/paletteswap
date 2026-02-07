# Palette Nested Colors Design

## Goal

Support nested colors in palette blocks using `color` as a reserved keyword. A palette block can be both a color (via its `color` attribute) and a namespace (via child attributes and nested blocks), to arbitrary depth.

## Example

```hcl
palette {
  black = "#000000"
  gray  = "#c0c0c0"

  highlight {
    color = palette.gray

    low  = "#21202e"
    mid  = "#403d52"
    high = "#524f67"

    deep {
      color = "#100f1a"
      muted = "#0a0a10"
    }
  }
}
```

- `palette.black` resolves to `#000000`
- `palette.highlight` resolves to `#c0c0c0` (from `color = palette.gray`)
- `palette.highlight.low` resolves to `#21202e`
- `palette.highlight.deep` resolves to `#100f1a`
- `palette.highlight.deep.muted` resolves to `#0a0a10`

## Design Decisions

- **`color` is reserved** in palette blocks. It defines the block's own color, not a child named "color".
- **Arbitrary nesting depth**. Any block can have `color` and/or sub-blocks.
- **No styling in palette**. Bold/italic/underline belong in syntax blocks only. Palette is purely colors.
- **Error at parse time** if a namespace-only block (no `color` attribute) is referenced as a color. Clear message: "palette.highlight is a group, not a color; add a `color` attribute or reference a specific child."
- **Source-order self-references**. Palette entries can reference earlier entries. Forward references are an error.

## Changes

### 1. New `color.Node` type

```go
type Node struct {
    Color    *Color
    Children map[string]*Node
}
```

Replaces `color.Tree` (`map[string]any`) for palette representation only. `Style` and `Tree` remain for syntax block use.

- `Color` is nil for namespace-only nodes (no `color` attribute).
- `Children` is nil for leaf nodes (flat color attributes).
- A node can have both `Color` and `Children` (the core feature).

### 2. Parser: rewrite `parsePaletteBody`

Current behavior: `isStyleBlock` treats `color` as meaning "this is a leaf." Blocks are either a `Style` or a `Tree`, never both.

New behavior for palette blocks:

1. If a block has a `color` attribute, parse it as the node's own color.
2. All other attributes are child leaf nodes.
3. All nested blocks are child nodes (recurse).
4. A block without `color` is a namespace-only node (`Color: nil`).

`isStyleBlock` no longer applies to palette parsing. It remains for syntax blocks.

### 3. Parser: source-order palette self-references

Currently the palette is parsed with a nil eval context — no self-references. Change to incremental parsing:

- Parse attributes and blocks in source order.
- After each entry is parsed, add it to the cty eval context.
- Later entries can reference earlier ones. Forward references produce an error.

This requires iterating the body's raw content by source position rather than using `body.Attributes` (which is unordered).

### 4. Parser: cty representation

`colorTreeToCty` (or a new `nodeToCity`) converts `*color.Node` to cty values:

- Leaf node (no children): `cty.StringVal(color.Hex())`
- Node with children: `cty.ObjectVal(...)` with `color` as a sibling key alongside children.

Add a `resolveColor(cty.Value) (string, error)` helper used at every consumption site:

- If the value is a `cty.String`, return it directly.
- If the value is a `cty.Object`, extract the `"color"` key and return that string.
- Otherwise, return an error.

This helper is used in:
- `decodeBodyToMap` (theme and ansi blocks)
- `parseSyntaxBody` (syntax referencing palette)
- `brighten()`/`darken()` function implementations

### 5. Engine: template rendering

- `templateData.Palette` changes from `color.Tree` to `*color.Node`.
- Add `getColorFromNode(node *Node, path []string) (Color, error)` — walks `Children` pointers, returns `Color` at the final node, errors if `Color == nil`.
- Update `resolveColorPath` palette case to use the new function.
- Remove palette from the `style` template function (palette has no styles).

### 6. Theme struct

- `theme.go`: `Palette` field changes from `color.Tree` to `*color.Node`.

## Out of Scope

- Syntax block changes. Syntax keeps its `style` reserved block and `color.Tree`/`color.Style` representation.
- Template format changes. Dot-notation paths (`hex "palette.highlight"`) work the same from the user's perspective.
