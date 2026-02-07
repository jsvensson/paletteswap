# pstheme-lsp Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an LSP server for `.pstheme` theme files providing diagnostics, completion, hover, go-to-definition, and color swatches.

**Architecture:** A standalone binary at `cmd/pstheme-lsp/` using `github.com/tliron/glsp` (protocol_3_16). A separate `internal/lsp/` package holds the analyzer (parses HCL from in-memory content, collects all diagnostics), document store, and handler logic. Reuses existing `internal/parser` and `internal/color` packages for palette resolution and color types.

**Tech Stack:** Go, `github.com/tliron/glsp` (protocol_3_16), `github.com/hashicorp/hcl/v2`, `github.com/zclconf/go-cty`

---

### Task 1: Project Setup and Minimal LSP Server

Add the glsp dependency and create a server binary that initializes, handles document sync, and shuts down cleanly.

**Files:**
- Create: `cmd/pstheme-lsp/main.go`
- Create: `internal/lsp/server.go`
- Create: `internal/lsp/documents.go`

**Step 1: Add glsp dependency**

Run: `go get github.com/tliron/glsp@latest && go get github.com/tliron/commonlog@latest`

**Step 2: Create the document store**

`internal/lsp/documents.go`:

```go
package lsp

import "sync"

// DocumentStore holds open document contents keyed by URI.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]string)}
}

func (s *DocumentStore) Open(uri, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = content
}

func (s *DocumentStore) Update(uri, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = content
}

func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func (s *DocumentStore) Get(uri string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.docs[uri]
	return content, ok
}
```

**Step 3: Create the server**

`internal/lsp/server.go`:

```go
package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
)

const serverName = "pstheme-lsp"

type Server struct {
	handler  protocol.Handler
	docs     *DocumentStore
	version  string
}

func NewServer(version string) *Server {
	s := &Server{
		docs:    NewDocumentStore(),
		version: version,
	}

	s.handler = protocol.Handler{
		Initialize:             s.initialize,
		Initialized:            s.initialized,
		Shutdown:               s.shutdown,
		SetTrace:               s.setTrace,
		TextDocumentDidOpen:    s.textDocumentDidOpen,
		TextDocumentDidChange:  s.textDocumentDidChange,
		TextDocumentDidClose:   s.textDocumentDidClose,
	}

	return s
}

func (s *Server) Run() error {
	commonlog.Configure(1, nil)
	srv := server.NewServer(&s.handler, serverName, false)
	return srv.RunStdio()
}

func (s *Server) initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := s.handler.CreateServerCapabilities()

	syncKind := protocol.TextDocumentSyncKindFull
	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: &protocol.True,
		Change:    &syncKind,
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &s.version,
		},
	}, nil
}

func (s *Server) initialized(_ *glsp.Context, _ *protocol.InitializedParams) error {
	return nil
}

func (s *Server) shutdown(_ *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func (s *Server) setTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *Server) textDocumentDidOpen(_ *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.docs.Open(string(params.TextDocument.URI), params.TextDocument.Text)
	return nil
}

func (s *Server) textDocumentDidChange(_ *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			s.docs.Update(string(params.TextDocument.URI), c.Text)
		}
	}
	return nil
}

func (s *Server) textDocumentDidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.docs.Close(string(params.TextDocument.URI))
	return nil
}
```

**Step 4: Create the main entry point**

`cmd/pstheme-lsp/main.go`:

```go
package main

import (
	"os"

	"github.com/jsvensson/paletteswap/internal/lsp"
)

var version = "dev"

func main() {
	s := lsp.NewServer(version)
	if err := s.Run(); err != nil {
		os.Exit(1)
	}
}
```

**Step 5: Verify it compiles**

Run: `go build ./cmd/pstheme-lsp/`
Expected: binary builds with no errors.

**Step 6: Run existing tests to check for regressions**

Run: `go test ./...`
Expected: all existing tests pass.

**Step 7: Commit**

Message: `feat: add pstheme-lsp skeleton with document sync`

---

### Task 2: Analyzer — HCL Parse and Diagnostic Collection

Build the analyzer that parses HCL from in-memory content and collects all diagnostics (syntax errors, semantic errors) without short-circuiting.

**Files:**
- Create: `internal/lsp/analyzer.go`
- Create: `internal/lsp/analyzer_test.go`

**Step 1: Write tests for the analyzer**

`internal/lsp/analyzer_test.go`:

```go
package lsp

import (
	"testing"
)

func TestAnalyze_ValidTheme(t *testing.T) {
	src := `
palette {
  base = "#191724"
  love = "#eb6f92"
}
theme {
  background = palette.base
}
ansi {
  black   = palette.base
  red     = palette.love
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", src)
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d:", len(result.Diagnostics))
		for _, d := range result.Diagnostics {
			t.Logf("  [%d:%d] %s", d.Range.Start.Line, d.Range.Start.Character, d.Message)
		}
	}
	if result.Palette == nil {
		t.Error("expected non-nil palette")
	}
}

func TestAnalyze_SyntaxError(t *testing.T) {
	src := `palette { base = }`
	result := Analyze("test.pstheme", src)
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for syntax error")
	}
}

func TestAnalyze_UndefinedPaletteRef(t *testing.T) {
	src := `
palette {
  base = "#191724"
}
theme {
  background = palette.nonexistent
}
ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", src)
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for undefined palette reference")
	}
}

func TestAnalyze_MissingANSI(t *testing.T) {
	src := `
palette {
  base = "#191724"
}
ansi {
  black = palette.base
}
`
	result := Analyze("test.pstheme", src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == DiagWarning {
			found = true
		}
	}
	if !found {
		t.Error("expected warning diagnostic for missing ANSI colors")
	}
}

func TestAnalyze_MissingPalette(t *testing.T) {
	src := `
meta {
  name = "test"
}
`
	result := Analyze("test.pstheme", src)
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostic for missing palette block")
	}
}

func TestAnalyze_InvalidHex(t *testing.T) {
	src := `
palette {
  bad = "not-a-color"
}
`
	result := Analyze("test.pstheme", src)
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostic for invalid hex color")
	}
}

func TestAnalyze_SymbolTable(t *testing.T) {
	src := `
palette {
  base = "#191724"
  highlight {
    low = "#21202e"
  }
}
ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", src)
	if _, ok := result.Symbols["palette.base"]; !ok {
		t.Error("expected symbol for palette.base")
	}
	if _, ok := result.Symbols["palette.highlight.low"]; !ok {
		t.Error("expected symbol for palette.highlight.low")
	}
}

func TestAnalyze_ColorLocations(t *testing.T) {
	src := `
palette {
  base = "#191724"
}
theme {
  background = palette.base
}
ansi {
  black   = "#000000"
  red     = "#ff0000"
  green   = "#00ff00"
  yellow  = "#ffff00"
  blue    = "#0000ff"
  magenta = "#ff00ff"
  cyan    = "#00ffff"
  white   = "#ffffff"
  bright_black   = "#808080"
  bright_red     = "#ff8080"
  bright_green   = "#80ff80"
  bright_yellow  = "#ffff80"
  bright_blue    = "#8080ff"
  bright_magenta = "#ff80ff"
  bright_cyan    = "#80ffff"
  bright_white   = "#ffffff"
}
`
	result := Analyze("test.pstheme", src)
	if len(result.Colors) == 0 {
		t.Error("expected color locations to be populated")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -v -count=1`
Expected: FAIL — `Analyze` not defined.

**Step 3: Implement the analyzer**

`internal/lsp/analyzer.go`:

The analyzer parses HCL from a string, walks the syntax tree, and collects:
- Diagnostics (all errors and warnings, never short-circuits)
- Palette `*color.Node` (for completion/hover)
- Symbol table: `map[string]Range` — palette paths to their definition positions
- Color locations: `[]ColorLocation` — every resolved color with its source range (for documentColor)

The implementation strategy:
1. Parse the HCL source with `hclsyntax.ParseConfig` — collect syntax diagnostics
2. Walk the body looking for palette, theme, ansi, syntax, meta blocks
3. Parse palette into `*color.Node` (reusing the incremental evaluation pattern from parser.parsePaletteBody) while recording symbol positions
4. Build eval context, then walk theme/ansi/syntax collecting semantic errors and color locations
5. Validate ANSI completeness (warning-level diagnostics)

This is the largest single file — approximately 300-400 lines. The key difference from `parser.Parse()` is that it never returns early and it records source positions for all resolved colors.

Use `protocol.Diagnostic` types directly from glsp for the diagnostic slice. Define type aliases for severity constants for convenience:

```go
var (
	DiagError   = protocol.DiagnosticSeverityError
	DiagWarning = protocol.DiagnosticSeverityWarning
	DiagInfo    = protocol.DiagnosticSeverityInformation
)
```

The `AnalysisResult` struct:

```go
type AnalysisResult struct {
	Diagnostics []protocol.Diagnostic
	Palette     *color.Node
	Symbols     map[string]protocol.Range  // "palette.highlight.low" → definition range
	Colors      []ColorLocation            // every resolved color with position
}

type ColorLocation struct {
	Range protocol.Range
	Color color.Color
}
```

Helper to convert `hcl.Pos` → `protocol.Position` (HCL is 1-based, LSP is 0-based):

```go
func hclPosToLSP(pos hcl.Pos) protocol.Position {
	return protocol.Position{
		Line:      uint32(pos.Line - 1),
		Character: uint32(pos.Column - 1),
	}
}

func hclRangeToLSP(r hcl.Range) protocol.Range {
	return protocol.Range{
		Start: hclPosToLSP(r.Start),
		End:   hclPosToLSP(r.End),
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -v -count=1`
Expected: PASS

**Step 5: Run all tests for regressions**

Run: `go test ./...`
Expected: all pass.

**Step 6: Commit**

Message: `feat: add LSP analyzer with diagnostics and symbol table`

---

### Task 3: Wire Diagnostics into Document Sync

Connect the analyzer to the document open/change handlers so diagnostics are published to the editor on every change.

**Files:**
- Modify: `internal/lsp/server.go`

**Step 1: Add an analyze-and-publish method to Server**

```go
func (s *Server) analyzeAndPublish(notify glsp.NotifyFunc, uri string) {
	content, ok := s.docs.Get(uri)
	if !ok {
		return
	}

	result := Analyze(uri, content)

	go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: result.Diagnostics,
	})
}
```

**Step 2: Call it from didOpen and didChange**

Update `textDocumentDidOpen` and `textDocumentDidChange` to call `s.analyzeAndPublish(context.Notify, uri)` after storing the document content.

**Step 3: Store the latest analysis result for other features**

Add a `results` map to `Server` (with a mutex) so completion/hover/definition/colors can use the latest analysis:

```go
type Server struct {
	handler  protocol.Handler
	docs     *DocumentStore
	version  string
	mu       sync.RWMutex
	results  map[string]*AnalysisResult
}
```

Update `analyzeAndPublish` to store the result, and add `Server.getResult(uri) *AnalysisResult` for readers.

**Step 4: Verify it compiles**

Run: `go build ./cmd/pstheme-lsp/`
Expected: builds cleanly.

**Step 5: Commit**

Message: `feat: wire diagnostics publishing to document sync`

---

### Task 4: Completion

Implement palette path completion, block-level completion, and function snippets.

**Files:**
- Create: `internal/lsp/completion.go`
- Create: `internal/lsp/completion_test.go`
- Modify: `internal/lsp/server.go` (register handler, add trigger character ".")

**Step 1: Write completion tests**

`internal/lsp/completion_test.go` — test the completion logic function directly (not through LSP protocol). Test cases:

- Cursor after `palette.` in a theme block → returns palette children (base, love, etc.)
- Cursor after `palette.highlight.` → returns nested children (low, mid, high)
- Cursor at attribute name position inside ansi block → returns missing ANSI color names
- Cursor at attribute name position inside style block → returns `color`, `bold`, `italic`, `underline` minus already-present

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -run TestCompletion -v -count=1`
Expected: FAIL

**Step 3: Implement completion**

`internal/lsp/completion.go`:

The completion function takes the analysis result, document content, and cursor position. It determines context by finding which block the cursor is in (walking the HCL syntax tree), then:

1. **Palette path**: if the text before cursor matches `palette.` (or `palette.something.`), walk the `*color.Node` tree and return children with `CompletionItemKindColor` and the hex value in `Detail`.
2. **ANSI names**: if cursor is in the `ansi` block at an attribute name position, offer required ANSI color names not yet defined.
3. **Style attributes**: if cursor is in a style block, offer `color`/`bold`/`italic`/`underline` minus present attributes.
4. **Top-level blocks**: if at root level, offer `meta`, `palette`, `theme`, `syntax`, `ansi`.
5. **Functions**: if at a value position, offer `brighten()` and `darken()` as snippet completions.

**Step 4: Register completion handler on Server**

In `server.go`, add `TextDocumentCompletion: s.textDocumentCompletion` to the handler, and set:

```go
capabilities.CompletionProvider = &protocol.CompletionOptions{
	TriggerCharacters: []string{"."},
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -run TestCompletion -v -count=1`
Expected: PASS

**Step 6: Run all tests**

Run: `go test ./...`
Expected: all pass.

**Step 7: Commit**

Message: `feat: add LSP completion for palette paths and block attributes`

---

### Task 5: Hover

Show resolved color info when hovering over palette references, hex literals, and function calls.

**Files:**
- Create: `internal/lsp/hover.go`
- Create: `internal/lsp/hover_test.go`
- Modify: `internal/lsp/server.go` (register handler)

**Step 1: Write hover tests**

`internal/lsp/hover_test.go` — test the hover logic function directly. Test cases:

- Hover over `palette.base` reference → returns markdown with hex, RGB, path
- Hover over `"#eb6f92"` hex literal → returns markdown with RGB
- Hover over `darken(palette.white, 0.2)` → returns markdown with computed color
- Hover over non-color text → returns nil

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -run TestHover -v -count=1`
Expected: FAIL

**Step 3: Implement hover**

`internal/lsp/hover.go`:

The hover function takes the analysis result, document content, and cursor position. It checks:

1. Is the cursor on a color location (from `AnalysisResult.Colors`)? If yes, format the resolved color as markdown: hex, RGB, and the source text.
2. Is the cursor on a hex literal? Parse it and show RGB breakdown.
3. Otherwise return nil.

Format as `protocol.MarkupContent` with `MarkupKindMarkdown`:

```
**palette.base**

`#191724` · `rgb(25, 23, 36)`
```

**Step 4: Register hover handler on Server**

Add `TextDocumentHover: s.textDocumentHover` to the handler.

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -run TestHover -v -count=1`
Expected: PASS

**Step 6: Commit**

Message: `feat: add LSP hover with color info`

---

### Task 6: Go to Definition

Navigate from palette references to their definitions in the palette block.

**Files:**
- Create: `internal/lsp/definition.go`
- Create: `internal/lsp/definition_test.go`
- Modify: `internal/lsp/server.go` (register handler)

**Step 1: Write definition tests**

`internal/lsp/definition_test.go` — test cases:

- Cursor on `palette.base` in theme block → returns location of `base = "#191724"` in palette
- Cursor on `palette.highlight.low` → returns location of `low = "#21202e"` in palette
- Cursor on a hex literal → returns nil (not a reference)
- Cursor on non-palette text → returns nil

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -run TestDefinition -v -count=1`
Expected: FAIL

**Step 3: Implement go-to-definition**

`internal/lsp/definition.go`:

The definition function takes the analysis result, document content, URI, and cursor position. It:

1. Identifies the word/expression at the cursor position
2. If it starts with `palette.`, extract the path (e.g. `palette.highlight.low` → `"palette.highlight.low"`)
3. Look up the path in `AnalysisResult.Symbols`
4. Return a `protocol.Location` with the URI and the symbol's definition range
5. Return nil if not on a palette reference

**Step 4: Register definition handler on Server**

Add `TextDocumentDefinition: s.textDocumentDefinition` to the handler.

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -run TestDefinition -v -count=1`
Expected: PASS

**Step 6: Commit**

Message: `feat: add LSP go-to-definition for palette references`

---

### Task 7: Document Colors and Color Presentation

Report all color values for inline swatches, and handle color picker writes.

**Files:**
- Create: `internal/lsp/colors.go`
- Create: `internal/lsp/colors_test.go`
- Modify: `internal/lsp/server.go` (register handlers)

**Step 1: Write color tests**

`internal/lsp/colors_test.go` — test cases:

- A valid theme with palette colors → `documentColor` returns ColorInformation for each hex literal and resolved reference
- Color values are correctly converted to LSP `protocol.Color` (float32 0.0–1.0)
- `colorPresentation` for a hex literal range → returns hex string replacement
- `colorPresentation` for a palette reference range → returns empty (no replacement offered)

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -run TestColor -v -count=1`
Expected: FAIL

**Step 3: Implement document colors**

`internal/lsp/colors.go`:

`documentColor` uses `AnalysisResult.Colors` to return `[]protocol.ColorInformation`. Convert `color.Color` to `protocol.Color`:

```go
func colorToLSP(c color.Color) protocol.Color {
	return protocol.Color{
		Red:   float32(c.R) / 255.0,
		Green: float32(c.G) / 255.0,
		Blue:  float32(c.B) / 255.0,
		Alpha: 1.0,
	}
}
```

`colorPresentation` takes the color and range from the client. Check if the range corresponds to a hex literal (by examining the document text at that range). If so, offer a replacement hex string. If the range is a palette reference, return an empty slice.

**Step 4: Register color handlers on Server**

Add `TextDocumentDocumentColor: s.textDocumentDocumentColor` and `TextDocumentColorPresentation: s.textDocumentColorPresentation` to the handler. Set `capabilities.ColorProvider = &protocol.True` in initialize.

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -run TestColor -v -count=1`
Expected: PASS

**Step 6: Run full test suite**

Run: `go test ./...`
Expected: all pass.

**Step 7: Commit**

Message: `feat: add LSP document colors and color presentation`

---

### Task 8: Manual Smoke Test

Verify the server works end-to-end with a real editor.

**Step 1: Build the binary**

Run: `go build -o pstheme-lsp ./cmd/pstheme-lsp/`

**Step 2: Create a test .pstheme file**

Copy `theme.hcl` to `test.pstheme` for manual testing.

**Step 3: Configure Neovim (or editor of choice)**

For Neovim, add to config:

```lua
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "*.pstheme",
  callback = function()
    vim.bo.filetype = "pstheme"
  end,
})

vim.lsp.start({
  name = "pstheme-lsp",
  cmd = { "/path/to/pstheme-lsp" },
  filetypes = { "pstheme" },
})
```

**Step 4: Verify features work**

- Open the `.pstheme` file — diagnostics should appear if there are errors
- Type `palette.` in a theme block — completion should offer palette colors
- Hover over a palette reference — should show hex and RGB
- Go-to-definition on a palette reference — should jump to palette block
- Color swatches should appear inline

**Step 5: Fix any issues discovered during smoke testing**

**Step 6: Commit any fixes**

Message: `fix: address issues found during smoke testing`
