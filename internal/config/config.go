package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
)

// Theme is the fully-resolved theme data, ready for template rendering.
type Theme struct {
	Meta    Meta
	Palette map[string]color.Color
	Theme   map[string]color.Color
	Syntax  color.ColorTree
	ANSI    map[string]color.Color
}

// Meta holds theme metadata.
type Meta struct {
	Name       string `hcl:"name,attr"`
	Author     string `hcl:"author,attr"`
	Appearance string `hcl:"appearance,attr"`
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
	palette map[string]color.Color
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

	// Decode palette entries from hcl.Body to map[string]string
	paletteStrings, err := decodeBodyToMap(raw.Palette.Entries, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing palette: %w", err)
	}

	palette, err := parseColorMap(paletteStrings)
	if err != nil {
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
func (l *Loader) Decode(target interface{}) error {
	if diags := gohcl.DecodeBody(l.body, l.ctx, target); diags.HasErrors() {
		return fmt.Errorf("decoding: %s", diags.Error())
	}
	return nil
}

// Palette returns the parsed palette colors.
func (l *Loader) Palette() map[string]color.Color {
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

// Load parses an HCL theme file and returns a fully-resolved Theme.
func Load(path string) (*Theme, error) {
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

	// Parse syntax manually (nested blocks with style properties)
	syntax, err := parseSyntax(resolved.Remain, loader.Context())
	if err != nil {
		return nil, fmt.Errorf("parsing syntax: %w", err)
	}

	meta := Meta{}
	if resolved.Meta != nil {
		meta = *resolved.Meta
	}

	return &Theme{
		Meta:    meta,
		Palette: loader.Palette(),
		Theme:   themeColors,
		Syntax:  syntax,
		ANSI:    ansiColors,
	}, nil
}

func parseMeta(body *hclsyntax.Body, theme *Theme) error {
	for _, block := range body.Blocks {
		if block.Type != "meta" {
			continue
		}
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return fmt.Errorf("parsing meta: %s", diags.Error())
		}
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating meta.%s: %s", name, diags.Error())
			}
			switch name {
			case "name":
				theme.Meta.Name = val.AsString()
			case "author":
				theme.Meta.Author = val.AsString()
			case "appearance":
				theme.Meta.Appearance = val.AsString()
			}
		}
		return nil
	}
	return nil
}

func parsePalette(body *hclsyntax.Body, theme *Theme) error {
	for _, block := range body.Blocks {
		if block.Type != "palette" {
			continue
		}
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return fmt.Errorf("parsing palette: %s", diags.Error())
		}
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating palette.%s: %s", name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("palette.%s: %w", name, err)
			}
			theme.Palette[name] = c
		}
		return nil
	}
	return fmt.Errorf("no palette block found")
}

func buildEvalContext(palette map[string]color.Color) *hcl.EvalContext {
	vals := make(map[string]cty.Value, len(palette))

	// Sort keys for deterministic output
	keys := make([]string, 0, len(palette))
	for k := range palette {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals[k] = cty.StringVal(palette[k].Hex())
	}
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": cty.ObjectVal(vals),
		},
	}
}

// parseColorBlock parses a flat block (theme or ansi) where all attributes
// reference palette colors.
func parseColorBlock(body *hclsyntax.Body, blockType string, ctx *hcl.EvalContext, dest map[string]color.Color) error {
	for _, block := range body.Blocks {
		if block.Type != blockType {
			continue
		}
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return fmt.Errorf("parsing %s: %s", blockType, diags.Error())
		}
		for name, attr := range attrs {
			val, diags := attr.Expr.Value(ctx)
			if diags.HasErrors() {
				return fmt.Errorf("evaluating %s.%s: %s", blockType, name, diags.Error())
			}
			c, err := color.ParseHex(val.AsString())
			if err != nil {
				return fmt.Errorf("%s.%s: %w", blockType, name, err)
			}
			dest[name] = c
		}
		return nil
	}
	return nil
}

// parseSyntaxBlock parses the syntax block, handling nested sub-blocks
// to build a ColorTree with dotted scope names.
func parseSyntaxBlock(body *hclsyntax.Body, ctx *hcl.EvalContext, dest color.ColorTree) error {
	for _, block := range body.Blocks {
		if block.Type != "syntax" {
			continue
		}
		return parseSyntaxBody(block.Body, ctx, dest)
	}
	return nil
}

// parseSyntax extracts and parses the syntax block from an hcl.Body.
// It handles the mixed structure (flat attributes + nested style blocks).
func parseSyntax(body hcl.Body, ctx *hcl.EvalContext) (color.ColorTree, error) {
	if body == nil {
		return make(color.ColorTree), nil
	}

	// The remain body contains unparsed blocks including syntax.
	// We need to find the syntax block within it.
	syntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// If not hclsyntax.Body, return empty tree (no syntax block)
		return make(color.ColorTree), nil
	}

	// Find the syntax block
	for _, block := range syntaxBody.Blocks {
		if block.Type == "syntax" {
			dest := make(color.ColorTree)
			if err := parseSyntaxBody(block.Body, ctx, dest); err != nil {
				return nil, err
			}
			return dest, nil
		}
	}

	return make(color.ColorTree), nil
}

func parseSyntaxBody(body *hclsyntax.Body, ctx *hcl.EvalContext, dest color.ColorTree) error {
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
			dest[attr.Name] = color.SyntaxStyle{Color: c}
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
			dest[name] = color.SyntaxStyle{Color: c}
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
			subtree := make(color.ColorTree)
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
func parseStyleBlock(body *hclsyntax.Body, ctx *hcl.EvalContext) (color.SyntaxStyle, error) {
	// Validate that all attributes are known (catches typos)
	knownAttrs := map[string]bool{"color": true, "bold": true, "italic": true, "underline": true}
	for name := range body.Attributes {
		if !knownAttrs[name] {
			return color.SyntaxStyle{}, fmt.Errorf("unknown attribute %q (valid: color, bold, italic, underline)", name)
		}
	}

	colorAttr, ok := body.Attributes["color"]
	if !ok {
		return color.SyntaxStyle{}, fmt.Errorf("missing required 'color' attribute")
	}

	val, diags := colorAttr.Expr.Value(ctx)
	if diags.HasErrors() {
		return color.SyntaxStyle{}, fmt.Errorf("evaluating color: %s", diags.Error())
	}

	c, err := color.ParseHex(val.AsString())
	if err != nil {
		return color.SyntaxStyle{}, fmt.Errorf("color: %w", err)
	}

	style := color.SyntaxStyle{Color: c}

	if attr, ok := body.Attributes["bold"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.SyntaxStyle{}, fmt.Errorf("evaluating bold: %s", diags.Error())
		}
		style.Bold = val.True()
	}

	if attr, ok := body.Attributes["italic"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.SyntaxStyle{}, fmt.Errorf("evaluating italic: %s", diags.Error())
		}
		style.Italic = val.True()
	}

	if attr, ok := body.Attributes["underline"]; ok {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return color.SyntaxStyle{}, fmt.Errorf("evaluating underline: %s", diags.Error())
		}
		style.Underline = val.True()
	}

	return style, nil
}
