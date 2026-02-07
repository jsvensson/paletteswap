# Palette Nested Colors Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support nested palette blocks where `color` is a reserved keyword that defines the block's own color, allowing a block to be both a color and a namespace.

**Architecture:** Add a `color.Node` type for palette representation. Rewrite palette parsing to produce `*Node` instead of `color.Tree`. Update cty conversion, engine path resolution, and all consumption sites. Syntax blocks are unchanged.

**Tech Stack:** Go 1.25, HCL v2 (hclsyntax), go-cty

---

### Task 1: Add `color.Node` type

**Files:**
- Modify: `internal/color/color.go`
- Test: `internal/color/color_test.go`

**Step 1: Write the failing test**

Add to `internal/color/color_test.go`:

```go
func TestNode_Lookup(t *testing.T) {
	// Build: palette { black = "#000000"; highlight { color = "#c0c0c0"; low = "#21202e" } }
	black, _ := ParseHex("#000000")
	gray, _ := ParseHex("#c0c0c0")
	low, _ := ParseHex("#21202e")

	root := &Node{
		Children: map[string]*Node{
			"black": {Color: &black},
			"highlight": {
				Color: &gray,
				Children: map[string]*Node{
					"low": {Color: &low},
				},
			},
		},
	}

	tests := []struct {
		name    string
		path    []string
		want    string
		wantErr bool
	}{
		{"flat leaf", []string{"black"}, "#000000", false},
		{"nested block with color", []string{"highlight"}, "#c0c0c0", false},
		{"nested child", []string{"highlight", "low"}, "#21202e", false},
		{"not found", []string{"missing"}, "", true},
		{"namespace only", []string{"nocolor"}, "", true},
	}

	// Add a namespace-only node for the error case
	root.Children["nocolor"] = &Node{
		Children: map[string]*Node{
			"child": {Color: &black},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := root.Lookup(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup(%v) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if err == nil && got.Hex() != tt.want {
				t.Errorf("Lookup(%v) = %q, want %q", tt.path, got.Hex(), tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/color/ -run TestNode_Lookup -v`
Expected: FAIL — `Node` type and `Lookup` method don't exist.

**Step 3: Write minimal implementation**

Add to `internal/color/color.go`:

```go
// Node represents a palette entry that can be both a color and a namespace.
// Color is nil for namespace-only nodes (groups without a color attribute).
// Children is nil for leaf nodes (flat color attributes).
type Node struct {
	Color    *Color
	Children map[string]*Node
}

// Lookup resolves a dot-path (as segments) to a Color.
// Returns an error if the path is not found or the target node has no color.
func (n *Node) Lookup(path []string) (Color, error) {
	current := n
	for _, part := range path {
		if current.Children == nil {
			return Color{}, fmt.Errorf("path not found: %s is a leaf, cannot traverse further", part)
		}
		child, ok := current.Children[part]
		if !ok {
			return Color{}, fmt.Errorf("path not found: %q does not exist", part)
		}
		current = child
	}
	if current.Color == nil {
		return Color{}, fmt.Errorf("path is a group, not a color; add a color attribute or reference a specific child")
	}
	return *current.Color, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/color/ -run TestNode_Lookup -v`
Expected: PASS

**Step 5: Commit**

Message: `feat: add color.Node type with Lookup method`

---

### Task 2: Add `resolveColor` cty helper

**Files:**
- Modify: `internal/parser/config.go`
- Test: `internal/parser/config_test.go`

**Step 1: Write the failing test**

Add to `internal/parser/config_test.go`:

```go
func TestResolveColor_String(t *testing.T) {
	val := cty.StringVal("#ff0000")
	got, err := resolveColor(val)
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
	got, err := resolveColor(val)
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
	_, err := resolveColor(val)
	if err == nil {
		t.Fatal("expected error for object without color key")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/parser/ -run TestResolveColor -v`
Expected: FAIL — `resolveColor` doesn't exist.

**Step 3: Write minimal implementation**

Add to `internal/parser/config.go`:

```go
// resolveColor extracts a color hex string from a cty.Value.
// If the value is a string, return it directly.
// If the value is an object, extract the "color" key.
func resolveColor(val cty.Value) (string, error) {
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -run TestResolveColor -v`
Expected: PASS

**Step 5: Commit**

Message: `feat: add resolveColor cty helper for nested palette values`

---

### Task 3: Add `nodeToCty` conversion

**Files:**
- Modify: `internal/parser/config.go`
- Test: `internal/parser/config_test.go`

**Step 1: Write the failing test**

Add to `internal/parser/config_test.go`:

```go
func TestNodeToCty_Leaf(t *testing.T) {
	c, _ := color.ParseHex("#ff0000")
	node := &color.Node{Color: &c}
	val := nodeToCty(node)
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
	val := nodeToCty(node)
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
	val := nodeToCty(node)
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/parser/ -run TestNodeToCty -v`
Expected: FAIL — `nodeToCty` doesn't exist.

**Step 3: Write minimal implementation**

Add to `internal/parser/config.go`:

```go
// nodeToCty converts a color.Node to a cty.Value for HCL evaluation context.
// Leaf nodes (no children) become cty.StringVal.
// Nodes with children become cty.ObjectVal, with "color" as a sibling key if the node has its own color.
func nodeToCty(node *color.Node) cty.Value {
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
		vals[k] = nodeToCty(node.Children[k])
	}

	return cty.ObjectVal(vals)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -run TestNodeToCty -v`
Expected: PASS

**Step 5: Commit**

Message: `feat: add nodeToCty conversion for palette Node`

---

### Task 4: Rewrite `parsePaletteBody` to produce `*color.Node`

This is the core change. The new parser iterates the body in source order, building the cty context incrementally so entries can reference earlier ones.

**Files:**
- Modify: `internal/parser/config.go`
- Test: `internal/parser/config_test.go`

**Step 1: Write the failing tests**

Add to `internal/parser/config_test.go`:

```go
func TestPaletteNestedColor(t *testing.T) {
	hcl := `
palette {
  gray = "#c0c0c0"

  highlight {
    color = palette.gray
    low   = "#21202e"
    high  = "#524f67"
  }
}

theme {
  background = palette.highlight
  surface    = palette.highlight.low
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// palette.highlight resolves to gray via the color keyword
	highlightColor, err := theme.Palette.Lookup([]string{"highlight"})
	if err != nil {
		t.Fatalf("Lookup(highlight) error: %v", err)
	}
	if highlightColor.Hex() != "#c0c0c0" {
		t.Errorf("palette.highlight = %q, want %q", highlightColor.Hex(), "#c0c0c0")
	}

	// palette.highlight.low resolves to child
	lowColor, err := theme.Palette.Lookup([]string{"highlight", "low"})
	if err != nil {
		t.Fatalf("Lookup(highlight.low) error: %v", err)
	}
	if lowColor.Hex() != "#21202e" {
		t.Errorf("palette.highlight.low = %q, want %q", lowColor.Hex(), "#21202e")
	}

	// theme.background resolves via palette.highlight -> color keyword
	bg := theme.Theme["background"]
	if bg.Hex() != "#c0c0c0" {
		t.Errorf("Theme[background] = %q, want %q", bg.Hex(), "#c0c0c0")
	}

	// theme.surface resolves via palette.highlight.low
	surface := theme.Theme["surface"]
	if surface.Hex() != "#21202e" {
		t.Errorf("Theme[surface] = %q, want %q", surface.Hex(), "#21202e")
	}
}

func TestPaletteDeepNesting(t *testing.T) {
	hcl := `
palette {
  highlight {
    color = "#c0c0c0"
    deep {
      color = "#100f1a"
      muted = "#0a0a10"
    }
  }
}

theme {
  a = palette.highlight
  b = palette.highlight.deep
  c = palette.highlight.deep.muted
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if theme.Theme["a"].Hex() != "#c0c0c0" {
		t.Errorf("a = %q, want %q", theme.Theme["a"].Hex(), "#c0c0c0")
	}
	if theme.Theme["b"].Hex() != "#100f1a" {
		t.Errorf("b = %q, want %q", theme.Theme["b"].Hex(), "#100f1a")
	}
	if theme.Theme["c"].Hex() != "#0a0a10" {
		t.Errorf("c = %q, want %q", theme.Theme["c"].Hex(), "#0a0a10")
	}
}

func TestPaletteNamespaceOnlyError(t *testing.T) {
	// Referencing a namespace-only block (no color attribute) as a color should error.
	hcl := `
palette {
  highlight {
    low = "#21202e"
  }
}

theme {
  background = palette.highlight
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error when referencing namespace-only block as color")
	}
}

func TestPaletteSelfReference(t *testing.T) {
	hcl := `
palette {
  base = "#191724"
  surface = palette.base
}

theme {
  background = palette.surface
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if theme.Theme["background"].Hex() != "#191724" {
		t.Errorf("background = %q, want %q", theme.Theme["background"].Hex(), "#191724")
	}
}

func TestPaletteForwardReferenceError(t *testing.T) {
	// Forward references within palette should error.
	hcl := `
palette {
  surface = palette.base
  base    = "#191724"
}

theme {
  background = palette.surface
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for forward reference in palette")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/parser/ -run "TestPaletteNestedColor|TestPaletteDeepNesting|TestPaletteNamespaceOnlyError|TestPaletteSelfReference|TestPaletteForwardReferenceError" -v`
Expected: FAIL — current parser doesn't support these features.

**Step 3: Write the implementation**

Replace `parsePaletteBody` and update `NewLoader` in `internal/parser/config.go`. The key changes:

1. **`parsePaletteBody`** now returns `*color.Node` instead of populating a `color.Tree`. It iterates attributes and blocks from the `hclsyntax.Body` in source order (sorted by position). For each:
   - If attribute named `color`: set as the node's own color
   - If other attribute: add as leaf child node
   - If block: recurse, add as child node

2. **Source-order parsing**: Collect all attributes and blocks, sort by source position, process sequentially. After each item is processed, rebuild the palette cty context so later items can reference earlier ones.

3. **`NewLoader`**: Call new `parsePaletteBody` which returns `*color.Node`. Use `nodeToCty` in `buildEvalContext`.

4. **`ParseResult.Palette`**: Change type from `color.Tree` to `*color.Node`.

5. **`buildEvalContext`**: Use `nodeToCty(palette)` instead of `colorTreeToCty(palette)`.

6. **`decodeBodyToMap`**: Use `resolveColor` instead of `val.AsString()` when evaluating attributes.

Here's the new `parsePaletteBody`:

```go
// paletteItem represents an attribute or block in source order.
type paletteItem struct {
	pos   hcl.Pos
	attr  *hclsyntax.Attribute // non-nil for attributes
	block *hclsyntax.Block     // non-nil for blocks
}

// parsePaletteBody parses a palette block body into a *color.Node.
// It processes items in source order, rebuilding the eval context after each
// so that later entries can reference earlier ones.
func parsePaletteBody(body *hclsyntax.Body, paletteRoot *color.Node) (*color.Node, error) {
	node := &color.Node{}

	// Collect all items and sort by source position
	var items []paletteItem
	for _, attr := range body.Attributes {
		items = append(items, paletteItem{pos: attr.SrcRange.Start, attr: attr})
	}
	for _, block := range body.Blocks {
		items = append(items, paletteItem{pos: block.DefRange().Start, block: block})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].pos.Line != items[j].pos.Line {
			return items[i].pos.Line < items[j].pos.Line
		}
		return items[i].pos.Column < items[j].pos.Column
	})

	for _, item := range items {
		// Rebuild eval context with current state of palette
		ctx := buildEvalContext(paletteRoot)

		if item.attr != nil {
			val, diags := item.attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return nil, fmt.Errorf("evaluating palette.%s: %s", item.attr.Name, diags.Error())
			}

			hexStr, err := resolveColor(val)
			if err != nil {
				return nil, fmt.Errorf("palette.%s: %w", item.attr.Name, err)
			}

			c, err := color.ParseHex(hexStr)
			if err != nil {
				return nil, fmt.Errorf("palette.%s: %w", item.attr.Name, err)
			}

			if item.attr.Name == "color" {
				// Reserved keyword: set this node's own color
				node.Color = &c
			} else {
				// Child leaf node
				if node.Children == nil {
					node.Children = make(map[string]*color.Node)
				}
				node.Children[item.attr.Name] = &color.Node{Color: &c}
			}
		} else {
			// Block: recurse
			childNode, err := parsePaletteBody(item.block.Body, paletteRoot)
			if err != nil {
				return nil, fmt.Errorf("palette.%s: %w", item.block.Type, err)
			}
			if node.Children == nil {
				node.Children = make(map[string]*color.Node)
			}
			node.Children[item.block.Type] = childNode
		}
	}

	return node, nil
}
```

Update `NewLoader` to use the new signature:

```go
func NewLoader(path string) (*Loader, error) {
	// ... (existing file reading and first-pass decode) ...

	paletteBody, ok := raw.Palette.Entries.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("palette block is not an hclsyntax.Body")
	}

	// Parse palette into Node tree
	// Start with an empty root node that gets populated incrementally
	palette := &color.Node{}
	var err error
	palette, err = parsePaletteBody(paletteBody, palette)
	if err != nil {
		return nil, fmt.Errorf("parsing palette: %w", err)
	}

	return &Loader{
		body:    file.Body,
		ctx:     buildEvalContext(palette),
		palette: palette,
	}, nil
}
```

Update `Loader` struct and `Palette()` method:

```go
type Loader struct {
	body    hcl.Body
	ctx     *hcl.EvalContext
	palette *color.Node
}

func (l *Loader) Palette() *color.Node {
	return l.palette
}
```

Update `ParseResult`:

```go
type ParseResult struct {
	Meta    Meta
	Palette *color.Node
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}
```

Update `buildEvalContext` to use `nodeToCty`:

```go
func buildEvalContext(palette *color.Node) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": nodeToCty(palette),
		},
		Functions: map[string]function.Function{
			"brighten": makeBrightenFunc(),
			"darken":   makeDarkenFunc(),
		},
	}
}
```

Update `decodeBodyToMap` to use `resolveColor`:

```go
func decodeBodyToMap(body hcl.Body, ctx *hcl.EvalContext) (map[string]string, error) {
	// ... (existing nil check and JustAttributes) ...
	for name, attr := range attrs {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("evaluating %s: %s", name, diags.Error())
		}
		hexStr, err := resolveColor(val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		result[name] = hexStr
	}
	return result, nil
}
```

Remove `colorTreeToCty` (replaced by `nodeToCty`).

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -v`
Expected: ALL parser tests pass, including the new ones.

**Step 5: Commit**

Message: `feat: rewrite palette parser to support nested colors with color keyword`

---

### Task 5: Update engine and theme for `*color.Node`

**Files:**
- Modify: `theme.go`
- Modify: `engine.go`
- Test: `engine_test.go`, `functions_test.go`, `path_test.go`

**Step 1: Update existing tests**

Tests that construct `Theme` with `color.Tree` for palette need updating to use `*color.Node`. Update `testTheme()` in `engine_test.go`:

```go
func testTheme() *Theme {
	base := color.Color{R: 25, G: 23, B: 36}
	love := color.Color{R: 235, G: 111, B: 146}
	highlightLow := color.Color{R: 33, G: 32, B: 46}
	highlightHigh := color.Color{R: 82, G: 79, B: 103}
	red := color.Color{R: 255, G: 0, B: 0}

	return &Theme{
		Meta: Meta{
			Name:       "Test Theme",
			Author:     "Tester",
			Appearance: "dark",
		},
		Palette: &color.Node{
			Children: map[string]*color.Node{
				"base": {Color: &base},
				"love": {Color: &love},
				"highlight": {
					Children: map[string]*color.Node{
						"low":  {Color: &highlightLow},
						"high": {Color: &highlightHigh},
					},
				},
				"custom": {
					Children: map[string]*color.Node{
						"bold": {Color: &red},
					},
				},
			},
		},
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
			"cursor":     {R: 235, G: 111, B: 146},
		},
		Syntax: color.Tree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
			"comment": color.Style{
				Color:  color.Color{R: 110, G: 106, B: 134},
				Italic: true,
			},
			"markup": color.Tree{
				"heading": color.Style{Color: color.Color{R: 235, G: 111, B: 146}},
				"bold": color.Style{
					Color: color.Color{R: 246, G: 193, B: 119},
					Bold:  true,
				},
			},
		},
		ANSI: map[string]color.Color{
			"black": {R: 0, G: 0, B: 0},
			"red":   {R: 235, G: 111, B: 146},
		},
	}
}
```

Update `TestTemplateFunctions_Hex` in `functions_test.go` — palette data:

```go
func TestTemplateFunctions_Hex(t *testing.T) {
	base := color.Color{R: 25, G: 23, B: 36}
	theme := &Theme{
		Palette: &color.Node{
			Children: map[string]*color.Node{
				"base": {Color: &base},
			},
		},
		// ... rest stays the same
	}
	// ...
}
```

Update `TestResolveColorPath_Palette` in `path_test.go`:

```go
func TestResolveColorPath_Palette(t *testing.T) {
	base := color.Color{R: 25, G: 23, B: 36}
	low := color.Color{R: 33, G: 32, B: 46}
	data := templateData{
		Palette: &color.Node{
			Children: map[string]*color.Node{
				"base": {Color: &base},
				"highlight": {
					Children: map[string]*color.Node{
						"low": {Color: &low},
					},
				},
			},
		},
	}
	// ... rest stays the same
}
```

Update `TestRunStyleFunc` in `engine_test.go` — the `style "palette.custom.bold"` template call. Since palette no longer uses `Style`, the `style` function should no longer support palette. Update the test to use syntax instead, or remove the palette style test.

Remove or update `TestRunStyleFunc` — palette no longer has styles:

```go
func TestRunStyleFunc(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `color={{ (style "syntax.comment").Color | hex }} italic={{ (style "syntax.comment").Italic }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "color=#6e6a86 italic=true"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

**Step 2: Run tests — they should fail (types don't match yet)**

Run: `go test ./... 2>&1`
Expected: FAIL — compilation errors from type mismatches.

**Step 3: Update `theme.go`**

```go
type Theme struct {
	Meta    Meta
	Palette *color.Node
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}
```

Update `Load()`:

```go
func Load(path string) (*Theme, error) {
	raw, err := parser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("loading theme: %w", err)
	}

	return &Theme{
		Meta: Meta{
			Name:       raw.Meta.Name,
			Author:     raw.Meta.Author,
			Appearance: raw.Meta.Appearance,
			URL:        raw.Meta.URL,
		},
		Palette: raw.Palette,
		Theme:   raw.Theme,
		Syntax:  raw.Syntax,
		ANSI:    raw.ANSI,
	}, nil
}
```

**Step 4: Update `engine.go`**

Change `templateData.Palette`:

```go
type templateData struct {
	Meta    Meta
	Palette *color.Node
	Theme   map[string]color.Color
	Syntax  color.Tree
	ANSI    map[string]color.Color
	FuncMap template.FuncMap
}
```

Replace `getStyleFromTree` usage for palette in `resolveColorPath` with `Node.Lookup`:

```go
case "palette":
	c, err := data.Palette.Lookup(rest)
	if err != nil {
		return color.Color{}, fmt.Errorf("palette path not found: %s (%w)", path, err)
	}
	return c, nil
```

Remove palette from `style` template function — only syntax is supported:

```go
"style": func(path string) (color.Style, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return color.Style{}, fmt.Errorf("invalid path %q", path)
	}

	block := parts[0]
	rest := parts[1:]

	switch block {
	case "syntax":
		return getStyleFromTree(data.Syntax, rest), nil
	default:
		return color.Style{}, fmt.Errorf("style only supports syntax block, got %q", block)
	}
},
```

**Step 5: Run all tests**

Run: `go test ./... -v`
Expected: ALL tests pass.

**Step 6: Commit**

Message: `feat: update engine and theme to use *color.Node for palette`

---

### Task 6: Update existing parser tests

Some existing parser tests use `color.Tree` type assertions on `ParseResult.Palette`. These need updating.

**Files:**
- Modify: `internal/parser/config_test.go`

**Step 1: Update `TestLoadPalette`**

```go
func TestLoadPalette(t *testing.T) {
	path := writeTempHCL(t, sampleHCL)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	// 6 top-level palette entries
	if len(theme.Palette.Children) != 6 {
		t.Errorf("len(Palette.Children) = %d, want 6", len(theme.Palette.Children))
	}
	loveColor, err := theme.Palette.Lookup([]string{"love"})
	if err != nil {
		t.Fatalf("Lookup(love) error: %v", err)
	}
	if loveColor.Hex() != "#eb6f92" {
		t.Errorf("palette.love = %q, want %q", loveColor.Hex(), "#eb6f92")
	}
}
```

**Step 2: Update `TestLoadNestedPalette`**

This test currently checks `color.Tree` and `color.Style` type assertions. Update to use `Lookup`:

```go
func TestLoadNestedPalette(t *testing.T) {
	hcl := `
palette {
  base = "#191724"

  highlight {
    low  = "#21202e"
    mid  = "#403d52"
    high = "#524f67"
  }
}

theme {
  background = palette.base
  cursor     = palette.highlight.high
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Check direct color
	baseColor, err := theme.Palette.Lookup([]string{"base"})
	if err != nil {
		t.Fatalf("Lookup(base) error: %v", err)
	}
	if baseColor.Hex() != "#191724" {
		t.Errorf("palette.base = %q, want %q", baseColor.Hex(), "#191724")
	}

	// Check nested color
	lowColor, err := theme.Palette.Lookup([]string{"highlight", "low"})
	if err != nil {
		t.Fatalf("Lookup(highlight.low) error: %v", err)
	}
	if lowColor.Hex() != "#21202e" {
		t.Errorf("palette.highlight.low = %q, want %q", lowColor.Hex(), "#21202e")
	}

	highColor, err := theme.Palette.Lookup([]string{"highlight", "high"})
	if err != nil {
		t.Fatalf("Lookup(highlight.high) error: %v", err)
	}
	if highColor.Hex() != "#524f67" {
		t.Errorf("palette.highlight.high = %q, want %q", highColor.Hex(), "#524f67")
	}

	// Check theme can reference nested palette values
	cursor := theme.Theme["cursor"]
	if cursor.Hex() != "#524f67" {
		t.Errorf("Theme[cursor] = %q, want %q", cursor.Hex(), "#524f67")
	}
}
```

Note: The old test had a `custom.bold` case with `Bold: true` — this was a style block in palette. Since palette no longer supports styles, remove that case from this test. It was already tested as a separate concern.

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: ALL tests pass.

**Step 4: Commit**

Message: `refactor: update parser tests to use color.Node API`

---

### Task 7: Clean up dead code

**Files:**
- Modify: `internal/parser/config.go`
- Modify: `internal/color/color.go` (if Tree is no longer used for palette)

**Step 1: Remove `colorTreeToCty` if no longer called**

Check if `colorTreeToCty` is still used anywhere (it may still be needed if syntax uses it). If syntax parsing still uses `buildEvalContext` with `colorTreeToCty` indirectly, keep it. Otherwise remove.

Actually, syntax doesn't go through `colorTreeToCty` — it's only used in `buildEvalContext` for the palette variable. Since we replaced that with `nodeToCty`, `colorTreeToCty` is dead code. Remove it.

**Step 2: Remove palette case from `isStyleBlock` usage in `parsePaletteBody`**

The old `parsePaletteBody` used `isStyleBlock`. The new one doesn't. Verify `isStyleBlock` is still used by syntax parsing. If yes, keep it. If no, remove it.

`isStyleBlock` is used in `parseSyntaxBody` (line 464), so keep it.

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: ALL tests pass.

**Step 4: Commit**

Message: `chore: remove dead colorTreeToCty function`

---

### Task 8: End-to-end verification with theme-example.hcl

**Files:**
- Read: `theme-example.hcl`

**Step 1: Run the CLI against the example theme**

Run: `go run ./cmd/paletteswap/ -t theme-example.hcl -d templates/ -o /tmp/paletteswap-test/`

Verify no errors. Check that the generated output correctly resolves:
- `palette.highlight` → `#c0c0c0` (the `color` value)
- `palette.highlight.low` → `#21202e`
- `palette.gray` → `#c0c0c0`

**Step 2: Commit if any template adjustments are needed**

This step may be a no-op if everything works.
