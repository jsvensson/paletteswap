# Extract Shared HCL Helpers into `internal/theme` — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate ~170 lines of duplicated HCL evaluation helpers between `parser/config.go` and `lsp/analyzer.go` by extracting them into a new `internal/theme` package.

**Architecture:** Create `internal/theme/theme.go` with the shared functions (`ResolveColor`, `NodeToCty`, `MakeBrightenFunc`, `MakeDarkenFunc`, `BuildEvalContext`, `RequiredANSIColors`), move the unit tests for those functions into `internal/theme/theme_test.go`, then update both `parser` and `lsp` packages to import and use the new package.

**Tech Stack:** Go, `hashicorp/hcl/v2`, `zclconf/go-cty`

---

### Task 1: Create `internal/theme/theme.go` with shared functions

**Files:**
- Create: `internal/theme/theme.go`

**Step 1: Create the file**

```go
package theme

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// RequiredANSIColors defines the 16 standard terminal colors that must be present.
var RequiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}

// ResolveColor extracts a color hex string from a cty.Value.
// If the value is a string, return it directly.
// If the value is an object, extract the "color" key.
func ResolveColor(val cty.Value) (string, error) {
	if val.Type() == cty.String {
		return val.AsString(), nil
	}
	if val.Type().IsObjectType() {
		if val.Type().HasAttribute("color") {
			colorVal := val.GetAttr("color")
			if colorVal.Type() == cty.String {
				return colorVal.AsString(), nil
			}
		}
		return "", fmt.Errorf("object has no 'color' attribute; reference a specific child or add a color attribute")
	}
	return "", fmt.Errorf("expected string or object with color attribute, got %s", val.Type().FriendlyName())
}

// NodeToCty converts a color.Node to a cty.Value for HCL evaluation context.
// Leaf nodes (no children) become cty.StringVal.
// Nodes with children become cty.ObjectVal, with "color" as a sibling key if the node has its own color.
func NodeToCty(node *color.Node) cty.Value {
	if node.Children == nil {
		// Leaf node: just a color string
		if node.Color != nil {
			return cty.StringVal(node.Color.Hex())
		}
		// Namespace-only leaf with no children — shouldn't happen, but handle gracefully
		return cty.EmptyObjectVal
	}

	vals := make(map[string]cty.Value, len(node.Children)+1)

	// Add the block's own color as "color" key
	if node.Color != nil {
		vals["color"] = cty.StringVal(node.Color.Hex())
	}

	// Add children
	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals[k] = NodeToCty(node.Children[k])
	}

	return cty.ObjectVal(vals)
}

// MakeBrightenFunc creates an HCL function that brightens a color.
// Usage: brighten("#hex", 0.1) or brighten(palette.color, 0.1)
func MakeBrightenFunc() function.Function {
	return function.New(&function.Spec{
		Description: "Brightens a color by the given percentage (-1.0 to 1.0)",
		Params: []function.Parameter{
			{
				Name: "color",
				Type: cty.String,
			},
			{
				Name: "percentage",
				Type: cty.Number,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			colorHex := args[0].AsString()
			pct, _ := args[1].AsBigFloat().Float64()

			c, err := color.ParseHex(colorHex)
			if err != nil {
				return cty.NilVal, err
			}

			brightened := color.Brighten(c, pct)
			return cty.StringVal(brightened.Hex()), nil
		},
	})
}

// MakeDarkenFunc creates an HCL function that darkens a color.
// Usage: darken("#hex", 0.1) or darken(palette.color, 0.1)
func MakeDarkenFunc() function.Function {
	return function.New(&function.Spec{
		Description: "Darkens a color by the given percentage (0.0 to 1.0)",
		Params: []function.Parameter{
			{
				Name: "color",
				Type: cty.String,
			},
			{
				Name: "percentage",
				Type: cty.Number,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			colorHex := args[0].AsString()
			pct, _ := args[1].AsBigFloat().Float64()

			c, err := color.ParseHex(colorHex)
			if err != nil {
				return cty.NilVal, err
			}

			darkened := color.Darken(c, pct)
			return cty.StringVal(darkened.Hex()), nil
		},
	})
}

// BuildEvalContext creates an HCL evaluation context with palette variables
// and brighten/darken functions.
func BuildEvalContext(palette *color.Node) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": NodeToCty(palette),
		},
		Functions: map[string]function.Function{
			"brighten": MakeBrightenFunc(),
			"darken":   MakeDarkenFunc(),
		},
	}
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/theme/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/theme/theme.go
git commit -m "refactor: create internal/theme package with shared HCL eval helpers"
```

---

### Task 2: Move unit tests for shared functions to `internal/theme`

**Files:**
- Create: `internal/theme/theme_test.go`
- Modify: `internal/parser/config_test.go` — remove lines 662-745 (the 6 tests for `nodeToCty` and `resolveColor`)

**Step 1: Create the test file**

Move the 6 unit tests from `config_test.go` that test `nodeToCty` and `resolveColor` directly. These are now exported functions in the `theme` package.

```go
package theme

import (
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeToCty_Leaf(t *testing.T) {
	c, _ := color.ParseHex("#ff0000")
	node := &color.Node{Color: &c}
	val := NodeToCty(node)
	if val.Type() != cty.String {
		t.Fatalf("expected string, got %s", val.Type().FriendlyName())
	}
	if val.AsString() != "#ff0000" {
		t.Errorf("got %q, want %q", val.AsString(), "#ff0000")
	}
}

func TestNodeToCty_NamespaceOnly(t *testing.T) {
	low, _ := color.ParseHex("#21202e")
	node := &color.Node{
		Children: map[string]*color.Node{
			"low": {Color: &low},
		},
	}
	val := NodeToCty(node)
	if !val.Type().IsObjectType() {
		t.Fatalf("expected object, got %s", val.Type().FriendlyName())
	}
	if val.GetAttr("low").AsString() != "#21202e" {
		t.Errorf("low = %q, want %q", val.GetAttr("low").AsString(), "#21202e")
	}
}

func TestNodeToCty_ColorAndChildren(t *testing.T) {
	gray, _ := color.ParseHex("#c0c0c0")
	low, _ := color.ParseHex("#21202e")
	node := &color.Node{
		Color: &gray,
		Children: map[string]*color.Node{
			"low": {Color: &low},
		},
	}
	val := NodeToCty(node)
	if !val.Type().IsObjectType() {
		t.Fatalf("expected object, got %s", val.Type().FriendlyName())
	}
	// "color" key holds the block's own color
	if val.GetAttr("color").AsString() != "#c0c0c0" {
		t.Errorf("color = %q, want %q", val.GetAttr("color").AsString(), "#c0c0c0")
	}
	if val.GetAttr("low").AsString() != "#21202e" {
		t.Errorf("low = %q, want %q", val.GetAttr("low").AsString(), "#21202e")
	}
}

func TestResolveColor_String(t *testing.T) {
	val := cty.StringVal("#ff0000")
	got, err := ResolveColor(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "#ff0000" {
		t.Errorf("got %q, want %q", got, "#ff0000")
	}
}

func TestResolveColor_ObjectWithColor(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"color": cty.StringVal("#c0c0c0"),
		"low":   cty.StringVal("#21202e"),
	})
	got, err := ResolveColor(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "#c0c0c0" {
		t.Errorf("got %q, want %q", got, "#c0c0c0")
	}
}

func TestResolveColor_ObjectWithoutColor(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"low": cty.StringVal("#21202e"),
	})
	_, err := ResolveColor(val)
	if err == nil {
		t.Fatal("expected error for object without color key")
	}
}
```

**Step 2: Remove the moved tests from `config_test.go`**

Remove lines 662-745 from `internal/parser/config_test.go` (the `TestNodeToCty_*` and `TestResolveColor_*` functions). Also remove the `"github.com/zclconf/go-cty/cty"` import if no longer used.

After removal, `config_test.go` should still have the `"github.com/zclconf/go-cty/cty"` import removed from the import block (verify no other test uses `cty` directly — if none do, remove it).

**Step 3: Verify tests pass**

Run: `go test ./internal/theme/ -v`
Expected: All 6 tests pass

Run: `go test ./internal/parser/ -v`
Expected: All remaining parser tests still pass

**Step 4: Commit**

```bash
git add internal/theme/theme_test.go internal/parser/config_test.go
git commit -m "refactor: move NodeToCty and ResolveColor tests to internal/theme"
```

---

### Task 3: Update `internal/parser/config.go` to use `internal/theme`

**Files:**
- Modify: `internal/parser/config.go`

**Step 1: Update imports**

Replace the import block (lines 3-15). Add `theme` import, remove `"sort"` (only used by `nodeToCty`) and `"github.com/zclconf/go-cty/cty/function"` (only used by function factories).

Before:
```go
import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)
```

After:
```go
import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/theme"
	"github.com/zclconf/go-cty/cty"
)
```

Note: `cty` is still needed — it's used by `decodeBodyToMap` at line 156 (`attr.Expr.Value`) and style parsing.

**Step 2: Remove `requiredANSIColors` (lines 17-23)**

Delete:
```go
// requiredANSIColors defines the 16 standard terminal colors that must be present.
var requiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}
```

**Step 3: Update `validateANSI` (references at lines 176 and 185)**

Replace `requiredANSIColors` with `theme.RequiredANSIColors` in both occurrences.

**Step 4: Update `NewLoader` (line 106)**

Replace `buildEvalContext(palette)` with `theme.BuildEvalContext(palette)`.

**Step 5: Update `decodeBodyToMap` (line 160)**

Replace `resolveColor(val)` with `theme.ResolveColor(val)`.

**Step 6: Update `parsePaletteBody` (lines 407 and 415)**

Replace:
- `buildEvalContext(paletteRoot)` → `theme.BuildEvalContext(paletteRoot)`
- `resolveColor(val)` → `theme.ResolveColor(val)`

**Step 7: Remove the 6 extracted functions (lines 253-378)**

Delete the following functions entirely:
- `resolveColor` (lines 253-270)
- `nodeToCty` (lines 272-304)
- `makeBrightenFunc` (lines 306-335)
- `makeDarkenFunc` (lines 337-366)
- `buildEvalContext` (lines 368-378)

**Step 8: Verify it compiles and tests pass**

Run: `go build ./internal/parser/`
Expected: No errors

Run: `go test ./internal/parser/ -v`
Expected: All tests pass

**Step 9: Commit**

```bash
git add internal/parser/config.go
git commit -m "refactor: update parser to use internal/theme for shared HCL helpers"
```

---

### Task 4: Update `internal/lsp/analyzer.go` to use `internal/theme`

**Files:**
- Modify: `internal/lsp/analyzer.go`

**Step 1: Update imports**

Replace the import block (lines 3-14). Add `theme` import, remove `"sort"` and `"github.com/zclconf/go-cty/cty/function"`.

Before:
```go
import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jsvensson/paletteswap/internal/color"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)
```

After:
```go
import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/theme"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/zclconf/go-cty/cty"
)
```

Note: `sort` is still needed — it's used by `analyzeBlock` at line 652.

**Step 2: Remove `requiredANSIColors` (lines 22-28)**

Delete the duplicated var declaration.

**Step 3: Update `BlockTypes` var (line 59)**

Replace `requiredANSIColors` with `theme.RequiredANSIColors` in the `"ansi"` block type.

**Step 4: Update `Analyze` function (lines 151-153, 161, 167)**

Replace:
- `makeBrightenFuncAnalyzer()` → `theme.MakeBrightenFunc()`
- `makeDarkenFuncAnalyzer()` → `theme.MakeDarkenFunc()`
- `nodeToCtyAnalyzer(palette)` → `theme.NodeToCty(palette)` (lines 161, 167)

**Step 5: Update `analyzePaletteBody` (lines 267, 284)**

Replace:
- `buildAnalyzerEvalContext(paletteRoot)` → `theme.BuildEvalContext(paletteRoot)`
- `analyzerResolveColor(val)` → `theme.ResolveColor(val)`

**Step 6: Update `analyzeColorBlock` (line 342)**

Replace `analyzerResolveColor(val)` → `theme.ResolveColor(val)`

**Step 7: Update `analyzeSyntaxBody` (line 384)**

Replace `analyzerResolveColor(val)` → `theme.ResolveColor(val)`

**Step 8: Update `validateANSICompleteness` (line 414)**

Replace `requiredANSIColors` → `theme.RequiredANSIColors`

**Step 9: Update `isValidANSIName` (line 574)**

Replace `requiredANSIColors` → `theme.RequiredANSIColors`

**Step 10: Update `processBlockAttribute` (line 712)**

Replace `analyzerResolveColor(val)` → `theme.ResolveColor(val)`

**Step 11: Update `buildBlockEvalContext` (line 780)**

Replace `nodeToCtyAnalyzer(node)` → `theme.NodeToCty(node)`

**Step 12: Remove the 6 extracted functions (lines 436-554)**

Delete the following functions entirely:
- `analyzerResolveColor` (lines 436-450)
- `nodeToCtyAnalyzer` (lines 465-492)
- `makeBrightenFuncAnalyzer` (lines 494-516)
- `makeDarkenFuncAnalyzer` (lines 518-540)
- `buildAnalyzerEvalContext` (lines 542-554)

Note: Keep `isReferenceExpr` (lines 452-463) — it is NOT duplicated in the parser, it's analyzer-only.

**Step 13: Verify it compiles and tests pass**

Run: `go build ./internal/lsp/`
Expected: No errors

Run: `go test ./internal/lsp/ -v`
Expected: All tests pass

**Step 14: Commit**

```bash
git add internal/lsp/analyzer.go
git commit -m "refactor: update LSP analyzer to use internal/theme for shared HCL helpers"
```

---

### Task 5: Update `internal/lsp/completion.go` to use `internal/theme`

**Files:**
- Modify: `internal/lsp/completion.go`

**Step 1: Add the import**

Add `"github.com/jsvensson/paletteswap/internal/theme"` to the import block.

**Step 2: Update `ansiCompletions` (line 274)**

Replace `requiredANSIColors` → `theme.RequiredANSIColors`

**Step 3: Verify it compiles and tests pass**

Run: `go test ./internal/lsp/ -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/lsp/completion.go
git commit -m "refactor: update LSP completion to use theme.RequiredANSIColors"
```

---

### Task 6: Final verification

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass across all packages

**Step 2: Build both binaries**

Run: `go build ./cmd/paletteswap/ && go build ./cmd/pstheme-lsp/`
Expected: Both build without errors

**Step 3: Verify `go vet` passes**

Run: `go vet ./...`
Expected: No issues
