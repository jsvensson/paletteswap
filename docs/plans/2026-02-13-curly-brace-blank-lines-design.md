# Design: Remove blank lines adjacent to curly braces

**Issue:** #64
**Date:** 2026-02-13

## Problem

The formatter should strip blank lines immediately after `{` and immediately before `}` in all block types (top-level and nested).

## Approach

Add two regex-based post-processing rules to `internal/format/format.go`, following the existing pattern used for collapsing multiple blank lines.

### Regexes

1. **Blank line after opening brace:** `{\n\s*\n` replaced with `{\n`
2. **Blank line before closing brace:** `\n\s*\n(\s*})` replaced with `\n$1` (preserving indentation)

### Ordering

Apply after the existing `multipleBlankLines` collapse step. This ensures multiple blank lines are first reduced to one, then any remaining single blank line adjacent to a brace is removed.

### Scope

All block types: meta, palette, theme, ansi, syntax, and nested blocks (highlight, markup, etc.).

## Test cases

- Blank line after opening brace
- Blank line before closing brace
- Both at once
- Nested blocks with blank lines adjacent to braces
