# pstheme-lsp Design

## Goal

Implement an LSP server for the `.pstheme` theme format, providing live diagnostics, completion, hover, go-to-definition, and inline color swatches. The server lives in this repo at `cmd/pstheme-lsp/` and reuses the existing `internal/parser` and `internal/color` packages. Editor-agnostic (Neovim, Zed, JetBrains all supported via standard LSP).

## Architecture

The LSP server is a standalone binary communicating over stdin/stdout using JSON-RPC 2.0. It uses `github.com/tliron/glsp` for protocol handling.

### Core Components

- **Server** -- handles LSP lifecycle (initialize, shutdown), document sync, dispatches requests to the analyzer.
- **Analyzer** -- takes document text, runs it through the HCL parser and palette resolver, produces a structured result: resolved palette tree, diagnostics list, symbol table, and color locations.
- **Document store** -- in-memory map of open file URIs to current text content, updated on `textDocument/didOpen` and `textDocument/didChange`.

### Reparse Strategy

Theme files are small (typically under 100 lines). On every document change, the analyzer re-parses the full file. No incremental parsing needed.

The analyzer is a separate code path from `parser.Parse()` -- it does not modify the existing parser. Instead, it walks the HCL syntax tree independently and collects all diagnostics rather than failing on the first error. This keeps the CLI behavior untouched.

## Feature 1: Diagnostics

Published via `textDocument/publishDiagnostics` on every document change.

### Reported Errors

- **HCL syntax errors** -- malformed blocks, unclosed braces, bad expressions. The HCL library provides these with source positions.
- **Undefined references** -- `palette.nonexistent` or `palette.highlight.nope`, caught during expression evaluation.
- **Missing ANSI colors** -- which of the 16 required colors are absent.
- **Missing palette block** -- required for a valid theme.
- **Unknown style attributes** -- anything other than `color`, `bold`, `italic`, `underline` in a style block.
- **Invalid hex values** -- `color.ParseHex` failures.
- **Function errors** -- bad arguments to `brighten()`/`darken()`.

### Position Mapping

The HCL library attaches `hcl.Range` (with start/end `hcl.Pos`) to every attribute, expression, and block. These map directly to LSP diagnostic positions.

## Feature 2: Completion

### Palette Path Completion

When the user types `palette.` inside a theme/syntax/ansi block, the server walks the resolved `*color.Node` tree and offers available children. Works recursively: `palette.highlight.` offers `low`, `mid`, `high`. Each completion item includes the resolved hex color in the detail field.

### Block-Level Completion

- Inside a style block: offer `color`, `bold`, `italic`, `underline` (filtering out already-present attributes).
- Inside `ansi`: offer the 16 required color names (filtering out already-defined ones).
- Top-level: offer `meta`, `palette`, `theme`, `syntax`, `ansi` block names.

### Function Completion

At attribute value positions, offer `brighten()` and `darken()` as snippet completions.

### Implementation

The server locates the cursor position in the HCL syntax tree, determines context (which block, attribute name vs. value position), and returns appropriate completions.

## Feature 3: Hover

### Palette References

Hovering over `palette.highlight.low` shows:
- Resolved hex value: `#21202e`
- RGB breakdown: `rgb(33, 32, 46)`
- Path: `palette.highlight.low`

### Hex Literals

Hovering over `"#eb6f92"` shows the RGB breakdown: `rgb(235, 111, 146)`.

### Function Calls

Hovering over `darken(palette.white, 0.2)` shows the resolved output color and the original input color for comparison.

### Style Blocks

Hovering over a style block (e.g. `comment`) shows the resolved color plus active style flags: `#6e6a86 italic`.

## Feature 4: Go to Definition

Cmd-click on a palette reference jumps to its definition in the palette block.

### Navigable References

- `palette.pine` in theme/syntax/ansi blocks jumps to `pine = "#31748f"` in the palette block.
- `palette.highlight.low` jumps to `low = "#21202e"` inside the `highlight` nested block.
- Self-references within the palette block (later entries referencing earlier ones).

### Symbol Table

During the analysis pass, the server builds a map from palette key paths to their definition `hcl.Range`. Go-to-definition resolves the reference at the cursor, looks up the path, and returns the stored location.

### Scope

Only palette definitions are navigation targets. Theme, ansi, and syntax entries are consumers, not definitions. Palette is the single source of truth.

## Feature 5: Color Presentation

Uses `textDocument/documentColor` to report color values at specific positions. Editors render these as inline swatches.

### Where Swatches Appear

- Hex literals in the palette block.
- Resolved references in theme/ansi/syntax blocks.
- Function results (e.g. `darken(palette.white, 0.2)` shows the computed color).

### Color Picker Behavior

`textDocument/colorPresentation` allows the editor's color picker to write back values. For hex literals, the server offers replacement with a new hex string. For palette references, no replacement is offered -- the picker should not overwrite `palette.pine` with a hex literal.

### Editor Support

Neovim, Zed, and JetBrains all support `documentColor` and render inline swatches.

## Project Structure

```
cmd/
  paletteswap/       # existing CLI
  pstheme-lsp/       # new LSP server binary
    main.go
internal/
  color/             # existing, reused
  parser/            # existing, reused
  lsp/               # new package
    server.go        # LSP server setup, handler registration
    analyzer.go      # document analysis, diagnostic collection
    completion.go    # completion logic
    hover.go         # hover logic
    definition.go    # go-to-definition logic
    colors.go        # documentColor / colorPresentation
    documents.go     # document store
```

## Dependencies

- `github.com/tliron/glsp` -- LSP protocol SDK (handler, protocol types, JSON-RPC server)
- Existing: `github.com/hashicorp/hcl/v2`, `github.com/zclconf/go-cty`

## File Extension

The format uses `.pstheme` as its file extension. The LSP server registers for this file type during initialization.
