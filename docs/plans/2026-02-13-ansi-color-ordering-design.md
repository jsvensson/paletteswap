# ANSI Color Ordering in Formatter

**Issue:** #62
**Date:** 2026-02-13

## Problem

The ANSI block must contain the 16 standard terminal colors in canonical order. The formatter should auto-fix misordered colors on normal invocation. With `--check`, it should report the file as needing formatting (existing behavior — `--check` compares pre/post).

## Canonical Order

Defined in `theme.RequiredANSIColors`:

```
black, red, green, yellow, blue, magenta, cyan, white,
bright_black, bright_red, bright_green, bright_yellow,
bright_blue, bright_magenta, bright_cyan, bright_white
```

## Approach

Hybrid: use `hclwrite.ParseConfig` for robust block identification, text-level reordering within the block to preserve comments.

### Pipeline

```
input → hclwrite.Format() → reorderANSIBlock() → regex post-processing → output
```

### reorderANSIBlock

1. Parse with `hclwrite.ParseConfig` to find the `ansi` block
2. If no ansi block or parse fails, return unchanged
3. Extract lines between braces
4. Group lines into entries: comment/blank lines + following attribute line
5. Map attribute name → entry text
6. Emit in canonical order
7. Splice back into source

### Dependencies

`format` → `theme` → `color` (no circular dependency)

## Testing

- Already-ordered stays unchanged
- Misordered gets reordered
- Comments above attributes travel with their attribute
- Inline comments preserved
- No ansi block → no change
- Partial/invalid HCL → graceful fallback
- Blank line separators between color groups
