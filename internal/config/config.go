package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/hashicorp/hcl/v2"
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
	Name       string
	Author     string
	Appearance string
}

// Load parses an HCL theme file and returns a fully-resolved Theme.
func Load(path string) (*Theme, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading theme file: %w", err)
	}

	file, diags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	body := file.Body.(*hclsyntax.Body)

	theme := &Theme{
		Palette: make(map[string]color.Color),
		Theme:   make(map[string]color.Color),
		Syntax:  make(color.ColorTree),
		ANSI:    make(map[string]color.Color),
	}

	// Parse meta block
	if err := parseMeta(body, theme); err != nil {
		return nil, err
	}

	// Parse palette block — must come first since other blocks reference it
	if err := parsePalette(body, theme); err != nil {
		return nil, err
	}

	// Build EvalContext with palette colors for resolving references
	ctx := buildEvalContext(theme.Palette)

	// Parse theme, syntax, and ansi blocks using the palette context
	if err := parseColorBlock(body, "theme", ctx, theme.Theme); err != nil {
		return nil, err
	}
	if err := parseSyntaxBlock(body, ctx, theme.Syntax); err != nil {
		return nil, err
	}
	if err := parseColorBlock(body, "ansi", ctx, theme.ANSI); err != nil {
		return nil, err
	}

	return theme, nil
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
			dest[attr.Name] = c
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
			dest[name] = c
		}
	}

	// Recurse into nested blocks
	for _, block := range body.Blocks {
		subtree := make(color.ColorTree)
		dest[block.Type] = subtree
		if err := parseSyntaxBody(block.Body, ctx, subtree); err != nil {
			return err
		}
	}

	return nil
}
