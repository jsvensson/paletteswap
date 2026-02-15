# Lightness Stepping Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add OKLCH lightness stepping to palette colors so a `transform { lightness { ... } }` block generates `l1`..`lN` children for every leaf color.

**Architecture:** Post-processing approach — palette is fully parsed first, then a second pass walks all leaf nodes, converts each to OKLCH, generates stepped children with absolute lightness values, and attaches them. The eval context is rebuilt with the expanded palette before other blocks are decoded.

**Tech Stack:** Go, `hashicorp/hcl/v2`, `go-cty`

---

### Task 1: Add OKLCH types and RGB conversion

**Files:**
- Create: `internal/color/oklch.go`
- Create: `internal/color/oklch_test.go`

**Step 1: Write the failing tests**

```go
// internal/color/oklch_test.go
package color

import (
	"math"
	"testing"
)

func TestRGBToOKLCH_KnownColors(t *testing.T) {
	tests := []struct {
		name    string
		color   Color
		wantL   float64
		wantC   float64
		wantH   float64
		tolerance float64
	}{
		{"black", Color{0, 0, 0}, 0.0, 0.0, 0.0, 0.001},
		{"white", Color{255, 255, 255}, 1.0, 0.0, 0.0, 0.001},
		{"red", Color{255, 0, 0}, 0.6279, 0.2577, 29.23, 0.01},
		{"green", Color{0, 128, 0}, 0.5196, 0.1766, 142.50, 0.6},
		{"blue", Color{0, 0, 255}, 0.4520, 0.3132, 264.05, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, c, h := RGBToOKLCH(tt.color)
			if math.Abs(l-tt.wantL) > tt.tolerance {
				t.Errorf("L = %f, want %f (±%f)", l, tt.wantL, tt.tolerance)
			}
			if math.Abs(c-tt.wantC) > tt.tolerance {
				t.Errorf("C = %f, want %f (±%f)", c, tt.wantC, tt.tolerance)
			}
			// Skip hue check for achromatic colors (C ≈ 0)
			if tt.wantC > 0.01 && math.Abs(h-tt.wantH) > tt.tolerance {
				t.Errorf("H = %f, want %f (±%f)", h, tt.wantH, tt.tolerance)
			}
		})
	}
}

func TestOKLCHToRGB_KnownColors(t *testing.T) {
	tests := []struct {
		name  string
		l, c, h float64
		want    Color
		tolerance uint8
	}{
		{"black", 0.0, 0.0, 0.0, Color{0, 0, 0}, 1},
		{"white", 1.0, 0.0, 0.0, Color{255, 255, 255}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OKLCHToRGB(tt.l, tt.c, tt.h)
			if absDiffUint8(got.R, tt.want.R) > tt.tolerance ||
				absDiffUint8(got.G, tt.want.G) > tt.tolerance ||
				absDiffUint8(got.B, tt.want.B) > tt.tolerance {
				t.Errorf("OKLCHToRGB(%f, %f, %f) = %v, want %v (±%d)",
					tt.l, tt.c, tt.h, got, tt.want, tt.tolerance)
			}
		})
	}
}

func TestRGBToOKLCH_Roundtrip(t *testing.T) {
	colors := []Color{
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{235, 111, 146},
		{49, 116, 143},
		{156, 207, 216},
	}

	for _, c := range colors {
		t.Run(c.Hex(), func(t *testing.T) {
			l, ch, h := RGBToOKLCH(c)
			got := OKLCHToRGB(l, ch, h)
			if absDiffUint8(got.R, c.R) > 1 || absDiffUint8(got.G, c.G) > 1 || absDiffUint8(got.B, c.B) > 1 {
				t.Errorf("roundtrip %v -> (%f,%f,%f) -> %v", c, l, ch, h, got)
			}
		})
	}
}

func absDiffUint8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/color/ -run TestRGBToOKLCH -v && go test ./internal/color/ -run TestOKLCHToRGB -v`
Expected: FAIL — functions not defined

**Step 3: Implement OKLCH conversion**

The conversion pipeline is: sRGB → linear RGB → OKLAB → OKLCH (and reverse).

```go
// internal/color/oklch.go
package color

import "math"

// RGBToOKLCH converts an RGB color to OKLCH (Lightness, Chroma, Hue).
// L is in [0, 1], C is in [0, ~0.37], H is in degrees [0, 360).
func RGBToOKLCH(c Color) (l, chroma, hue float64) {
	// sRGB to linear RGB
	r := srgbToLinear(float64(c.R) / 255.0)
	g := srgbToLinear(float64(c.G) / 255.0)
	b := srgbToLinear(float64(c.B) / 255.0)

	// Linear RGB to OKLAB
	l_, a, bLab := linearRGBToOKLAB(r, g, b)

	// OKLAB to OKLCH
	chroma = math.Sqrt(a*a + bLab*bLab)
	hue = 0.0
	if chroma > 1e-8 {
		hue = math.Atan2(bLab, a) * (180.0 / math.Pi)
		if hue < 0 {
			hue += 360.0
		}
	}

	return l_, chroma, hue
}

// OKLCHToRGB converts OKLCH values to an RGB color.
// L is in [0, 1], C is in [0, ~0.37], H is in degrees [0, 360).
func OKLCHToRGB(l, chroma, hue float64) Color {
	// OKLCH to OKLAB
	hRad := hue * (math.Pi / 180.0)
	a := chroma * math.Cos(hRad)
	b := chroma * math.Sin(hRad)

	// OKLAB to linear RGB
	r, g, bLinear := oklabToLinearRGB(l, a, b)

	// Linear RGB to sRGB, clamp to [0, 1]
	r = clamp01(linearToSRGB(r))
	g = clamp01(linearToSRGB(g))
	bLinear = clamp01(linearToSRGB(bLinear))

	return Color{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(bLinear * 255.0)),
	}
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

func linearRGBToOKLAB(r, g, b float64) (l, a, bLab float64) {
	// RGB to LMS (using Oklab M1 matrix)
	l_ := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	// Cube root
	l_ = math.Cbrt(l_)
	m = math.Cbrt(m)
	s = math.Cbrt(s)

	// LMS to Lab (using Oklab M2 matrix)
	l = 0.2104542553*l_ + 0.7936177850*m - 0.0040720468*s
	a = 1.9779984951*l_ - 2.4285922050*m + 0.4505937099*s
	bLab = 0.0259040371*l_ + 0.7827717662*m - 0.8086757660*s

	return l, a, bLab
}

func oklabToLinearRGB(l, a, bLab float64) (r, g, b float64) {
	// Lab to LMS (inverse of M2)
	l_ := l + 0.3963377774*a + 0.2158037573*bLab
	m := l - 0.1055613458*a - 0.0638541728*bLab
	s := l - 0.0894841775*a - 1.2914855480*bLab

	// Cube
	l_ = l_ * l_ * l_
	m = m * m * m
	s = s * s * s

	// LMS to linear RGB (inverse of M1)
	r = +4.0767416621*l_ - 3.3077115913*m + 0.2309699292*s
	g = -1.2684380046*l_ + 2.6097574011*m - 0.3413193965*s
	b = -0.0041960863*l_ - 0.7034186147*m + 1.7076147010*s

	return r, g, b
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/color/ -run "TestRGBToOKLCH|TestOKLCHToRGB" -v`
Expected: PASS

**Step 5: Commit**

```
feat(color): add OKLCH color space conversion

Implements RGB ↔ OKLCH conversion via the linear sRGB → OKLAB → OKLCH
pipeline. Foundation for lightness stepping in #78.
```

---

### Task 2: Add StepLightness function

**Files:**
- Modify: `internal/color/oklch.go`
- Modify: `internal/color/oklch_test.go`

**Step 1: Write the failing test**

```go
// Append to internal/color/oklch_test.go

func TestStepLightness(t *testing.T) {
	// A mid-gray color stepped to a specific lightness
	gray := Color{128, 128, 128}

	tests := []struct {
		name      string
		color     Color
		lightness float64
	}{
		{"low lightness", gray, 0.3},
		{"mid lightness", gray, 0.6},
		{"high lightness", gray, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StepLightness(tt.color, tt.lightness)

			// Verify the output has the requested lightness
			gotL, _, _ := RGBToOKLCH(got)
			if math.Abs(gotL-tt.lightness) > 0.02 {
				t.Errorf("StepLightness(%v, %f) produced L=%f, want L≈%f",
					tt.color, tt.lightness, gotL, tt.lightness)
			}
		})
	}
}

func TestStepLightness_PreservesHueChroma(t *testing.T) {
	// A saturated color — hue and chroma should be approximately preserved
	red := Color{255, 0, 0}
	_, origC, origH := RGBToOKLCH(red)

	stepped := StepLightness(red, 0.8)
	_, gotC, gotH := RGBToOKLCH(stepped)

	// Chroma may be clipped by gamut, but hue should be very close
	if math.Abs(gotH-origH) > 1.0 {
		t.Errorf("hue shifted: orig=%f, got=%f", origH, gotH)
	}
	// Chroma should be in the same ballpark (may decrease for gamut clipping)
	if gotC > origC*1.1 {
		t.Errorf("chroma increased unexpectedly: orig=%f, got=%f", origC, gotC)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/color/ -run TestStepLightness -v`
Expected: FAIL — function not defined

**Step 3: Implement StepLightness**

```go
// Append to internal/color/oklch.go

// StepLightness returns a new Color with the given absolute OKLCH lightness,
// preserving the original color's hue and chroma. Lightness should be in [0, 1].
func StepLightness(c Color, lightness float64) Color {
	_, chroma, hue := RGBToOKLCH(c)
	return OKLCHToRGB(lightness, chroma, hue)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/color/ -run TestStepLightness -v`
Expected: PASS

**Step 5: Commit**

```
feat(color): add StepLightness for absolute OKLCH lightness

Sets a color to a specific OKLCH lightness while preserving hue and
chroma. Used by the palette transform to generate lightness variants.
```

---

### Task 3: Add ApplyLightnessSteps to Node tree

**Files:**
- Modify: `internal/color/color.go` (add method to Node)
- Modify: `internal/color/color_test.go` (add tests)

**Step 1: Write the failing test**

```go
// Append to internal/color/color_test.go

func TestApplyLightnessSteps_FlatLeaf(t *testing.T) {
	c, _ := ParseHex("#808080")
	root := &Node{
		Children: map[string]*Node{
			"gray": {Color: &c},
		},
	}

	ApplyLightnessSteps(root, 0.3, 0.9, 3)

	// gray should still have its original color
	if root.Children["gray"].Color == nil {
		t.Fatal("expected gray to retain its color")
	}
	if root.Children["gray"].Color.Hex() != "#808080" {
		t.Errorf("gray.Color = %q, want %q", root.Children["gray"].Color.Hex(), "#808080")
	}

	// gray should now have l1, l2, l3 children
	if root.Children["gray"].Children == nil {
		t.Fatal("expected gray to have children after stepping")
	}
	for _, name := range []string{"l1", "l2", "l3"} {
		child, ok := root.Children["gray"].Children[name]
		if !ok {
			t.Errorf("expected child %q", name)
			continue
		}
		if child.Color == nil {
			t.Errorf("%s has nil color", name)
		}
	}
}

func TestApplyLightnessSteps_Nested(t *testing.T) {
	mid, _ := ParseHex("#403d52")
	root := &Node{
		Children: map[string]*Node{
			"highlight": {
				Children: map[string]*Node{
					"mid": {Color: &mid},
				},
			},
		},
	}

	ApplyLightnessSteps(root, 0.4, 0.8, 2)

	// highlight.mid should have l1, l2
	midNode := root.Children["highlight"].Children["mid"]
	if midNode.Children == nil {
		t.Fatal("expected mid to have children")
	}
	if _, ok := midNode.Children["l1"]; !ok {
		t.Error("expected l1")
	}
	if _, ok := midNode.Children["l2"]; !ok {
		t.Error("expected l2")
	}
}

func TestApplyLightnessSteps_PreservesOriginalColor(t *testing.T) {
	c, _ := ParseHex("#eb6f92")
	root := &Node{
		Children: map[string]*Node{
			"love": {Color: &c},
		},
	}

	ApplyLightnessSteps(root, 0.5, 0.9, 3)

	// Original color still accessible
	got, err := root.Lookup([]string{"love"})
	if err != nil {
		t.Fatalf("Lookup(love) error: %v", err)
	}
	if got.Hex() != "#eb6f92" {
		t.Errorf("love = %q, want %q", got.Hex(), "#eb6f92")
	}
}

func TestApplyLightnessSteps_SkipsNamespaceOnly(t *testing.T) {
	child, _ := ParseHex("#000000")
	root := &Node{
		Children: map[string]*Node{
			"group": {
				// No Color — namespace only
				Children: map[string]*Node{
					"inner": {Color: &child},
				},
			},
		},
	}

	ApplyLightnessSteps(root, 0.3, 0.9, 3)

	// group should NOT get l1/l2/l3 (it has no color)
	// but group.inner SHOULD
	if _, ok := root.Children["group"].Children["l1"]; ok {
		t.Error("namespace-only group should not get lightness steps")
	}
	if root.Children["group"].Children["inner"].Children == nil {
		t.Fatal("expected inner to have children")
	}
	if _, ok := root.Children["group"].Children["inner"].Children["l1"]; !ok {
		t.Error("expected inner.l1")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/color/ -run TestApplyLightnessSteps -v`
Expected: FAIL — function not defined

**Step 3: Implement ApplyLightnessSteps**

```go
// Add to internal/color/color.go

// ApplyLightnessSteps walks the node tree and generates l1..lN children for every
// leaf color node. Each step gets an evenly-spaced absolute OKLCH lightness value
// between low and high. The original color is preserved as the node's own Color.
func ApplyLightnessSteps(node *Node, low, high float64, steps int) {
	if steps < 1 {
		return
	}
	applyLightnessStepsRecursive(node, low, high, steps)
}

func applyLightnessStepsRecursive(node *Node, low, high float64, steps int) {
	if node.Children != nil {
		for _, child := range node.Children {
			applyLightnessStepsRecursive(child, low, high, steps)
		}
		return
	}

	// Leaf node with a color — generate stepped children
	if node.Color == nil {
		return
	}

	node.Children = make(map[string]*Node, steps)
	for i := 0; i < steps; i++ {
		var lightness float64
		if steps == 1 {
			lightness = (low + high) / 2.0
		} else {
			lightness = low + (high-low)*float64(i)/float64(steps-1)
		}
		stepped := StepLightness(*node.Color, lightness)
		name := fmt.Sprintf("l%d", i+1)
		node.Children[name] = &Node{Color: &stepped}
	}
}
```

Note: This requires adding `"fmt"` to the import in `color.go` (it's already there).

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/color/ -run TestApplyLightnessSteps -v`
Expected: PASS

**Step 5: Commit**

```
feat(color): add ApplyLightnessSteps for palette transform

Walks the Node tree and generates l1..lN children on every leaf color
node with evenly-spaced absolute OKLCH lightness values.
```

---

### Task 4: Parse the transform block in the parser

**Files:**
- Modify: `internal/parser/config.go`
- Modify: `internal/parser/config_test.go`

**Step 1: Write the failing test**

```go
// Append to internal/parser/config_test.go

func TestPaletteTransformLightness(t *testing.T) {
	hcl := `
palette {
  transform {
    lightness {
      range = [0.4, 0.8]
      steps = 3
    }
  }

  base = "#808080"
}

theme {
  bg_l1 = palette.base.l1
  bg_l2 = palette.base.l2
  bg_l3 = palette.base.l3
  bg    = palette.base
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Original color still accessible
	base, err := theme.Palette.Lookup([]string{"base"})
	if err != nil {
		t.Fatalf("Lookup(base) error: %v", err)
	}
	if base.Hex() != "#808080" {
		t.Errorf("palette.base = %q, want %q", base.Hex(), "#808080")
	}

	// Stepped colors accessible
	for _, name := range []string{"l1", "l2", "l3"} {
		_, err := theme.Palette.Lookup([]string{"base", name})
		if err != nil {
			t.Errorf("Lookup(base.%s) error: %v", name, err)
		}
	}

	// Theme can reference stepped colors
	if _, ok := theme.Theme["bg_l1"]; !ok {
		t.Error("expected theme.bg_l1")
	}
	if _, ok := theme.Theme["bg_l2"]; !ok {
		t.Error("expected theme.bg_l2")
	}
}

func TestPaletteTransformLightnessNested(t *testing.T) {
	hcl := `
palette {
  transform {
    lightness {
      range = [0.5, 0.9]
      steps = 2
    }
  }

  highlight {
    mid = "#403d52"
  }
}

theme {
  a = palette.highlight.mid.l1
  b = palette.highlight.mid.l2
}
` + completeANSI
	path := writeTempHCL(t, hcl)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if _, ok := theme.Theme["a"]; !ok {
		t.Error("expected theme.a (palette.highlight.mid.l1)")
	}
	if _, ok := theme.Theme["b"]; !ok {
		t.Error("expected theme.b (palette.highlight.mid.l2)")
	}
}

func TestPaletteNoTransform(t *testing.T) {
	// Palette without transform should work as before
	path := writeTempHCL(t, sampleHCL)
	theme, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// No stepped children should exist
	base, err := theme.Palette.Lookup([]string{"base"})
	if err != nil {
		t.Fatalf("Lookup(base) error: %v", err)
	}
	if base.Hex() != "#191724" {
		t.Errorf("palette.base = %q, want %q", base.Hex(), "#191724")
	}

	// base should be a leaf node (no children)
	if theme.Palette.Children["base"].Children != nil {
		t.Error("expected base to be a leaf node without transform")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/parser/ -run "TestPaletteTransform|TestPaletteNoTransform" -v`
Expected: FAIL — transform block causes parse error (unexpected block)

**Step 3: Implement transform block parsing**

The `transform` block needs to be extracted from the palette body before processing colors. Since `parsePaletteBody` iterates over `body.Blocks`, we need to filter out the `transform` block and handle it separately.

Add a `Transform` type and parsing logic to `internal/parser/config.go`:

```go
// Add to internal/parser/config.go

// LightnessTransform holds the configuration for lightness stepping.
type LightnessTransform struct {
	Low   float64
	High  float64
	Steps int
}

// parseTransformBlock extracts and parses the transform block from a palette body.
// Returns nil if no transform block is present.
func parseTransformBlock(body *hclsyntax.Body) (*LightnessTransform, error) {
	for _, block := range body.Blocks {
		if block.Type != "transform" {
			continue
		}

		// Find lightness sub-block
		for _, sub := range block.Body.Blocks {
			if sub.Type != "lightness" {
				continue
			}

			rangeAttr, ok := sub.Body.Attributes["range"]
			if !ok {
				return nil, fmt.Errorf("transform.lightness: missing 'range' attribute")
			}
			stepsAttr, ok := sub.Body.Attributes["steps"]
			if !ok {
				return nil, fmt.Errorf("transform.lightness: missing 'steps' attribute")
			}

			rangeVal, diags := rangeAttr.Expr.Value(nil)
			if diags.HasErrors() {
				return nil, fmt.Errorf("transform.lightness.range: %s", diags.Error())
			}
			stepsVal, diags := stepsAttr.Expr.Value(nil)
			if diags.HasErrors() {
				return nil, fmt.Errorf("transform.lightness.steps: %s", diags.Error())
			}

			if !rangeVal.Type().IsTupleType() || rangeVal.LengthInt() != 2 {
				return nil, fmt.Errorf("transform.lightness.range: must be a two-element list")
			}

			low, _ := rangeVal.Index(cty.NumberIntVal(0)).AsBigFloat().Float64()
			high, _ := rangeVal.Index(cty.NumberIntVal(1)).AsBigFloat().Float64()
			steps, _ := stepsVal.AsBigFloat().Int64()

			if steps < 1 {
				return nil, fmt.Errorf("transform.lightness.steps: must be at least 1")
			}

			return &LightnessTransform{
				Low:   low,
				High:  high,
				Steps: int(steps),
			}, nil
		}

		return nil, nil // transform block exists but no lightness sub-block
	}

	return nil, nil // no transform block
}
```

Then update `parsePaletteBody` to skip transform blocks, and update `NewLoader` to apply the transform after parsing:

In `parsePaletteBody`, add at the start of the block loop:

```go
// In the block processing part of parsePaletteBody, skip transform blocks:
// Change the block handling in parsePaletteBody to skip "transform" blocks
```

In `NewLoader`, after building the palette and before building the eval context:

```go
// After parsePaletteBody succeeds, parse transform and apply:
transform, err := parseTransformBlock(paletteBody)
if err != nil {
    return nil, fmt.Errorf("parsing palette transform: %w", err)
}

if transform != nil {
    color.ApplyLightnessSteps(palette, transform.Low, transform.High, transform.Steps)
}
```

The key changes to `parsePaletteBody`:
- Skip blocks with type `"transform"` when iterating items
- This prevents the transform block from being treated as a nested palette group

The key changes to `NewLoader`:
- After `parsePaletteBody` returns, call `parseTransformBlock` on the body
- If a transform is found, call `color.ApplyLightnessSteps`
- Then build the eval context (which will include the stepped children)

Also need to add `"github.com/zclconf/go-cty/cty"` to the imports in `config.go`.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/ -run "TestPaletteTransform|TestPaletteNoTransform" -v`
Expected: PASS

Also run ALL existing tests to verify nothing broke:
Run: `go test ./internal/parser/ -v`
Expected: All PASS

**Step 5: Commit**

```
feat(parser): parse transform block and apply lightness stepping

Extracts the optional transform { lightness { ... } } block from
palette, applies OKLCH lightness stepping to all leaf colors, and
rebuilds the eval context with stepped children. Closes #78.
```

---

### Task 5: Add transform support to LSP analyzer

**Files:**
- Modify: `internal/lsp/analyzer.go`
- Modify: `internal/lsp/analyzer_test.go`

**Step 1: Write the failing test**

```go
// Append to internal/lsp/analyzer_test.go

func TestAnalyze_PaletteTransformLightness(t *testing.T) {
	content := `
palette {
  transform {
    lightness {
      range = [0.4, 0.8]
      steps = 3
    }
  }

  base = "#808080"
}

theme {
  bg    = palette.base
  bg_l1 = palette.base.l1
  bg_l3 = palette.base.l3
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
	result := Analyze("test.pstheme", content)

	// Should have no errors
	for _, d := range result.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			t.Errorf("unexpected error: %s", d.Message)
		}
	}

	// Palette should have stepped children
	if result.Palette == nil {
		t.Fatal("expected non-nil palette")
	}
	baseNode, ok := result.Palette.Children["base"]
	if !ok {
		t.Fatal("expected 'base' in palette")
	}
	if baseNode.Children == nil {
		t.Fatal("expected 'base' to have stepped children")
	}
	for _, name := range []string{"l1", "l2", "l3"} {
		if _, ok := baseNode.Children[name]; !ok {
			t.Errorf("expected child %q in palette.base", name)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/lsp/ -run TestAnalyze_PaletteTransformLightness -v`
Expected: FAIL — transform block causes analyzer errors

**Step 3: Implement transform in analyzer**

In `analyzer.go`, after palette is processed (around line 153-155), add transform parsing and application. The analyzer uses `hclsyntax.Body` directly, so we can reuse the `parseTransformBlock` function from the parser package, or extract it to a shared location.

Best approach: move `parseTransformBlock` and `LightnessTransform` to the `theme` package (since it's shared infrastructure), or keep it in `parser` and import it from `lsp`. Since `lsp` doesn't currently import `parser`, adding it to `theme` is cleaner.

Actually, looking at the architecture, the simplest approach is to put `parseTransformBlock` and `LightnessTransform` in the `parser` package and export them, then import from `lsp`. But `lsp` doesn't import `parser` currently.

Alternative: put it in the `color` package alongside `ApplyLightnessSteps` — but that would create a dependency on HCL syntax from the color package, which is wrong.

Best approach: put `LightnessTransform` in `color` (it's just a data struct) and `parseTransformBlock` in `parser` (it does HCL parsing). In the LSP analyzer, duplicate the parsing logic or extract to a shared helper in `theme`.

Simplest: put `ParseTransformBlock` in the `parser` package as an exported function, and have `lsp` import `parser`. Check if this creates a circular import — `parser` imports `theme` and `color`, `lsp` imports `theme` and `color`. Adding `lsp` → `parser` should be fine (no cycle).

In `analyzer.go`, after palette processing:

```go
// After palette processing (around line 154):
// Apply transform if present
if paletteBody, ok := blockBodies["palette"]; ok {
    transform, err := parser.ParseTransformBlock(paletteBody)
    if err != nil {
        result.addError(hcl.Range{}, err.Error())
    } else if transform != nil {
        color.ApplyLightnessSteps(result.Palette, transform.Low, transform.High, transform.Steps)
    }
    ctx.Variables["palette"] = theme.NodeToCty(result.Palette)
}
```

Wait — but `ctx.Variables["palette"]` is already set on line 154. We need to re-set it after applying steps. The palette processing and transform application should be:

```go
if paletteBody, ok := blockBodies["palette"]; ok {
    palette, _ := result.analyzeBlock(paletteBody, BlockTypes["palette"], ctx, "palette", nil)
    result.Palette = palette

    // Apply lightness transform if present
    transform, err := parser.ParseTransformBlock(paletteBody)
    if err != nil {
        result.addError(hcl.Range{}, err.Error())
    } else if transform != nil {
        color.ApplyLightnessSteps(palette, transform.Low, transform.High, transform.Steps)
    }

    ctx.Variables["palette"] = theme.NodeToCty(palette)
}
```

Also need to update `analyzeBlock` to skip `transform` blocks in the palette (just like the parser skips them). Add a check in the block collection loop:

```go
// In analyzeBlock, when collecting blocks, skip "transform":
for _, block := range body.Blocks {
    if block.Type == "transform" {
        continue // handled separately for palette lightness stepping
    }
    // ... existing block processing
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/lsp/ -run TestAnalyze_PaletteTransformLightness -v`
Expected: PASS

Also run ALL tests:
Run: `go test ./... -v`
Expected: All PASS

**Step 5: Commit**

```
feat(lsp): support transform block in palette analysis

The LSP analyzer now parses the transform { lightness { ... } } block
and applies OKLCH lightness stepping to palette colors, making stepped
variants available for completion and validation.
```

---

### Task 6: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Manual smoke test with theme file**

Update `theme-example.pstheme` to include a transform block and verify it works with the parser. A quick test:

```go
// In a temporary test or via CLI: verify that a theme file with transform
// can be parsed and stepped colors are accessible
```

**Step 3: Final commit if any fixups needed**

---

## Summary of files changed

| File | Action | Purpose |
|------|--------|---------|
| `internal/color/oklch.go` | Create | OKLCH types, RGB↔OKLCH conversion, StepLightness |
| `internal/color/oklch_test.go` | Create | Tests for OKLCH conversion and StepLightness |
| `internal/color/color.go` | Modify | Add ApplyLightnessSteps |
| `internal/color/color_test.go` | Modify | Tests for ApplyLightnessSteps |
| `internal/parser/config.go` | Modify | Parse transform block, apply stepping in NewLoader |
| `internal/parser/config_test.go` | Modify | Integration tests for transform |
| `internal/lsp/analyzer.go` | Modify | Apply transform in LSP analysis |
| `internal/lsp/analyzer_test.go` | Modify | LSP analyzer transform test |
