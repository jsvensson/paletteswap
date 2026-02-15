# Lightness Stepping Design

GitHub: #78 (sub-issue of #76)

## Summary

Add OKLCH-based lightness stepping to palette colors. A `transform` block inside `palette` defines a lightness range and step count. After the palette is parsed, a post-processing pass generates `l1`..`lN` child nodes for every leaf color, each with the same hue and chroma but different lightness values.

## HCL Syntax

```hcl
palette {
  transform {
    lightness {
      range = [0.725, 0.925]
      steps = 5
    }
  }

  black = "#000000"
  highlight {
    mid = "#403d52"
  }
}
```

- `transform` block is optional; palette works as before without it
- `lightness` is a nested block within `transform` (extensible for future chroma/hue transforms)
- `range` — two-element list of absolute OKLCH lightness bounds (0.0–1.0)
- `steps` — number of evenly-spaced steps across the range

## Output Shape

Every leaf color node gains `l1`..`lN` children. The original color remains accessible at its original path via the existing `color` key mechanism:

- `palette.black` → original `#000000`
- `palette.black.l1` → lightness = 0.725
- `palette.black.l5` → lightness = 0.925
- `palette.highlight.mid` → original `#403d52`
- `palette.highlight.mid.l1` through `palette.highlight.mid.l5` → stepped variants

Step naming: `l` prefix, 1-indexed.

## Architecture

### Color package (`internal/color/`)

- Add OKLCH type and RGB ↔ OKLCH conversion (linear sRGB → OKLAB → OKLCH pipeline)
- Add `StepLightness(c Color, lightness float64) Color` — returns a new Color with the given absolute OKLCH lightness, preserving hue and chroma

### Color Node tree (`internal/color/`)

- Add `ApplyLightnessSteps(node *Node, low, high float64, steps int)` — walks the tree, finds leaf nodes (nodes with a color but no children), generates `l1`..`lN` children. The leaf's color is preserved as the node's own color (accessible via the `color` key in `NodeToCty`).

### Parser (`internal/parser/`)

- Parse the `transform` block from the palette body before processing colors
- After building the palette tree, call `ApplyLightnessSteps` if transform was present
- Rebuild eval context with the expanded palette before decoding other blocks

### LSP analyzer (`internal/lsp/`)

- Same flow: after palette analysis, apply lightness steps, rebuild eval context

## Implementation approach

Post-processing (Approach A): the palette is fully parsed first, then a second pass generates stepped children. This cleanly separates parsing from transform logic. Stepped colors are available in theme/syntax/ansi blocks but not for self-referencing within the palette block itself.

## Testing

- Unit tests for RGB ↔ OKLCH round-trip conversion with known color values
- Unit tests for `StepLightness` verifying correct lightness with hue/chroma preserved
- Unit tests for `ApplyLightnessSteps` tree walking (leaf nodes get children, nested nodes work, original color preserved)
- Integration test: parse a `.pstheme` with a transform block, verify stepped colors accessible in theme/syntax blocks
- LSP analyzer test: verify stepped palette colors produce correct symbols and color locations

## Future work

- Per-color transform overrides (#79)
- Chroma and hue transform blocks under `transform`
