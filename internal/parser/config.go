package parser

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

// requiredANSIColors defines the 16 standard terminal colors that must be present.
var requiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}

// ParseResult holds the raw parsed theme data.
type ParseResult struct {
	Meta    Meta
	Palette color.Tree
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}

// Meta holds theme metadata.
type Meta struct {
	Name       string `hcl:"name,optional"`
	Author     string `hcl:"author,optional"`
	Appearance string `hcl:"appearance,optional"`
	URL        string `hcl:"url,optional"`
}

// PaletteBlock wraps a single palette block for gohcl decoding.
type PaletteBlock struct {
	Entries hcl.Body `hcl:",remain"`
}

// RawConfig captures the palette block first (no EvalContext needed).
type RawConfig struct {
	Palette *PaletteBlock `hcl:"palette,block"`
	Remain  hcl.Body      `hcl:",remain"`
}

// ColorBlock wraps a block with arbitrary color attributes for gohcl decoding.
type ColorBlock struct {
	Entries hcl.Body `hcl:",remain"`
}

// ResolvedConfig decodes blocks that reference palette.
type ResolvedConfig struct {
	Meta   *Meta       `hcl:"meta,block"`
	Theme  *ColorBlock `hcl:"theme,block"`
	ANSI   *ColorBlock `hcl:"ansi,block"`
	Remain hcl.Body    `hcl:",remain"` // captures syntax for manual parsing
}

// Loader handles two-pass HCL decoding with palette resolution.
type Loader struct {
	body    hcl.Body
	ctx     *hcl.EvalContext
	palette color.Tree
}

// NewLoader parses an HCL file and builds the evaluation context from palette.
func NewLoader(path string) (*Loader, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading theme file: %w", err)
	}

	file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	// First pass: extract palette (literal values, no context needed)
	var raw RawConfig
	if diags := gohcl.DecodeBody(file.Body, nil, &raw); diags.HasErrors() {
		return nil, fmt.Errorf("decoding palette: %s", diags.Error())
	}

	if raw.Palette == nil {
		return nil, fmt.Errorf("no palette block found")
	}

	// Parse nested palette structure (supports direct colors, style blocks, and nested scopes)
	paletteBody, ok := raw.Palette.Entries.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("palette block is not an hclsyntax.Body")
	}

	palette := make(color.Tree)
	if err := parsePaletteBody(paletteBody, nil, palette); err != nil {
		return nil, fmt.Errorf("parsing palette: %w", err)
	}

	return &Loader{
		body:    file.Body,
		ctx:     buildEvalContext(palette),
		palette: palette,
	}, nil
}

// Decode decodes a value using the palette context.
// Reusable for any blocks that reference palette values.
func (l *Loader) Decode(target any) error {
	if diags := gohcl.DecodeBody(l.body, l.ctx, target); diags.HasErrors() {
		return fmt.Errorf("decoding: %s", diags.Error())
	}
	return nil
}

// Palette returns the parsed palette colors.
func (l *Loader) Palette() color.Tree {
	return l.palette
}

// Context returns the EvalContext for manual parsing.
func (l *Loader) Context() *hcl.EvalContext {
	return l.ctx
}

// parseColorMap converts a map of hex strings to a map of Colors.
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

// decodeBodyToMap decodes an hcl.Body with arbitrary string attributes into a map.
func decodeBodyToMap(body hcl.Body, ctx *hcl.EvalContext) (map[string]string, error) {
	if body == nil {
		return make(map[string]string), nil
	}

	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return nil, fmt.Errorf("getting attributes: %s", diags.Error())
	}

	result := make(map[string]string, len(attrs))
	for name, attr := range attrs {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("evaluating %s: %s", name, diags.Error())
		}
		result[name] = val.AsString()
	}
	return result, nil
}

// validateANSI checks that all 16 required ANSI colors are present.
func validateANSI(ansi map[string]color.Color) error {
	if len(ansi) == 0 {
		return fmt.Errorf("ansi block incomplete: no colors defined")
	}

	var missing []string
	for _, colorName := range requiredANSIColors {
		if _, ok := ansi[colorName]; !ok {
			missing = append(missing, colorName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("ansi block incomplete\nMissing colors: %s\nRequired colors: %s",
			strings.Join(missing, ", "),
			strings.Join(requiredANSIColors, ", "))
	}

	return nil
}

// Parse parses an HCL theme file and returns a fully-resolved ParseResult.
func Parse(path string) (*ParseResult, error) {
	loader, err := NewLoader(path)
	if err != nil {
		return nil, err
	}

	// Second pass: decode blocks that reference palette
	var resolved ResolvedConfig
	if err := loader.Decode(&resolved); err != nil {
		return nil, err
	}

	// Convert ColorBlock entries to color maps
	var themeStrings map[string]string
	if resolved.Theme != nil {
		themeStrings, err = decodeBodyToMap(resolved.Theme.Entries, loader.Context())
		if err != nil {
			return nil, fmt.Errorf("parsing theme: %w", err)
		}
	}
	themeColors, err := parseColorMap(themeStrings)
	if err != nil {
		return nil, fmt.Errorf("parsing theme: %w", err)
	}

	var ansiStrings map[string]string
	if resolved.ANSI != nil {
		ansiStrings, err = decodeBodyToMap(resolved.ANSI.Entries, loader.Context())
		if err != nil {
			return nil, fmt.Errorf("parsing ansi: %w", err)
		}
	}
	ansiColors, err := parseColorMap(ansiStrings)
	if err != nil {
		return nil, fmt.Errorf("parsing ansi: %w", err)
	}

	if err := validateANSI(ansiColors); err != nil {
		return nil, err
	}

	// Parse syntax manually (nested blocks with style properties)
	syntax, err := parseSyntax(resolved.Remain, loader.Context())
	if err != nil {
		return nil, fmt.Errorf("parsing syntax: %w", err)
	}

	meta := Meta{}
	if resolved.Meta != nil {
		meta = *resolved.Meta
	}

	return &ParseResult{
		Meta:    meta,
		Palette: loader.Palette(),
		Theme:   themeColors,
		Syntax:  syntax,
		ANSI:    ansiColors,
	}, nil
}

// colorTreeToCty converts a color.Tree to a cty.Value for HCL evaluation context.
func colorTreeToCty(tree color.Tree) cty.Value {
	vals := make(map[string]cty.Value, len(tree))

	// Sort keys for deterministic output
	keys := make([]string, 0, len(tree))
	for k := range tree {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := tree[k]
		if style, ok := v.(color.Style); ok {
			vals[k] = cty.StringVal(style.Color.Hex())
		} else if subtree, ok := v.(color.Tree); ok {
			vals[k] = colorTreeToCty(subtree)
		}
	}

	return cty.ObjectVal(vals)
}

// makeBrightenFunc creates an HCL function that brightens a color.
// Usage: brighten("#hex", 0.1) or brighten(palette.color, 0.1)
func makeBrightenFunc() function.Function {
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

func buildEvalContext(palette color.Tree) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": colorTreeToCty(palette),
		},
		Functions: map[string]function.Function{
			"brighten": makeBrightenFunc(),
		},
	}
}

// parsePaletteBody parses a palette block body with support for:
// - Direct color attributes: key = "#hex"
// - Style blocks: key { color = "#hex", bold = true }
// - Nested blocks: key { sub = ... }
// Palette is parsed without context (no palette references allowed within palette).
func parsePaletteBody(body *hclsyntax.Body, ctx *hcl.EvalContext, dest color.Tree) error {
	// Parse attributes at this level (direct color assignments)
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		// JustAttributes fails if there are blocks; use manual iteration instead
		for _, attr := range body.Attributes {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating palette attribute %s: %s", attr.Name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("palette.%s: %w", attr.Name, err)
			}
			dest[attr.Name] = color.Style{Color: c}
		}
	} else {
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating palette.%s: %s", name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("palette.%s: %w", name, err)
			}
			dest[name] = color.Style{Color: c}
		}
	}

	// Recurse into nested blocks
	for _, block := range body.Blocks {
		if isStyleBlock(block.Body) {
			style, err := parseStyleBlock(block.Body, ctx)
			if err != nil {
				return fmt.Errorf("palette.%s: %w", block.Type, err)
			}
			dest[block.Type] = style
		} else {
			subtree := make(color.Tree)
			dest[block.Type] = subtree
			if err := parsePaletteBody(block.Body, ctx, subtree); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseSyntax extracts and parses the syntax block from an hcl.Body.
// It handles the mixed structure (flat attributes + nested style blocks).
func parseSyntax(body hcl.Body, ctx *hcl.EvalContext) (color.Tree, error) {
	if body == nil {
		return make(color.Tree), nil
	}

	// The remain body contains unparsed blocks including syntax.
	// We need to find the syntax block within it.
	syntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// If not hclsyntax.Body, return empty tree (no syntax block)
		return make(color.Tree), nil
	}

	// Find the syntax block
	for _, block := range syntaxBody.Blocks {
		if block.Type == "syntax" {
			dest := make(color.Tree)
			if err := parseSyntaxBody(block.Body, ctx, dest); err != nil {
				return nil, err
			}
			return dest, nil
		}
	}

	return make(color.Tree), nil
}

func parseSyntaxBody(body *hclsyntax.Body, ctx *hcl.EvalContext, dest color.Tree) error {
	// Parse attributes at this level
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		// JustAttributes fails if there are blocks; use manual iteration instead
		for _, attr := range body.Attributes {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating syntax attribute %s: %s", attr.Name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("syntax.%s: %w", attr.Name, err)
			}
			dest[attr.Name] = color.Style{Color: c}
		}
	} else {
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating syntax.%s: %s", name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("syntax.%s: %w", name, err)
			}
			dest[name] = color.Style{Color: c}
		}
	}

	// Recurse into nested blocks
	for _, block := range body.Blocks {
		if isStyleBlock(block.Body) {
			style, err := parseStyleBlock(block.Body, ctx)
			if err != nil {
				return fmt.Errorf("syntax.%s: %w", block.Type, err)
			}
			dest[block.Type] = style
		} else {
			subtree := make(color.Tree)
			dest[block.Type] = subtree
			if err := parseSyntaxBody(block.Body, ctx, subtree); err != nil {
				return err
			}
		}
	}

	return nil
}

// isStyleBlock returns true if the body contains a "color" attribute,
// indicating it is a style block rather than a nested scope.
func isStyleBlock(body *hclsyntax.Body) bool {
	_, hasColor := body.Attributes["color"]
	return hasColor
}

// parseStyleBlock parses a style block with a required "color" attribute
// and optional "bold", "italic", "underline" boolean attributes.
func parseStyleBlock(body *hclsyntax.Body, ctx *hcl.EvalContext) (color.Style, error) {
	// Validate that all attributes are known (catches typos)
	knownAttrs := map[string]bool{"color": true, "bold": true, "italic": true, "underline": true}
	for name := range body.Attributes {
		if !knownAttrs[name] {
			return color.Style{}, fmt.Errorf("unknown attribute %q (valid: color, bold, italic, underline)", name)
		}
	}

	colorAttr, ok := body.Attributes["color"]
	if !ok {
		return color.Style{}, fmt.Errorf("missing required 'color' attribute")
	}

	val, diags := colorAttr.Expr.Value(ctx)
	if diags.HasErrors() {
		return color.Style{}, fmt.Errorf("evaluating color: %s", diags.Error())
	}

	c, err := color.ParseHex(val.AsString())
	if err != nil {
		return color.Style{}, fmt.Errorf("color: %w", err)
	}

	style := color.Style{Color: c}

	if attr, ok := body.Attributes["bold"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.Style{}, fmt.Errorf("evaluating bold: %s", diags.Error())
		}
		style.Bold = val.True()
	}

	if attr, ok := body.Attributes["italic"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.Style{}, fmt.Errorf("evaluating italic: %s", diags.Error())
		}
		style.Italic = val.True()
	}

	if attr, ok := body.Attributes["underline"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.Style{}, fmt.Errorf("evaluating underline: %s", diags.Error())
		}
		style.Underline = val.True()
	}

	return style, nil
}
