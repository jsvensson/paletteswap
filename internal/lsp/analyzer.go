package lsp

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

var (
	DiagError   = protocol.DiagnosticSeverityError
	DiagWarning = protocol.DiagnosticSeverityWarning
	DiagInfo    = protocol.DiagnosticSeverityInformation
)

// requiredANSIColors defines the 16 standard terminal colors that must be present.
var requiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}

// AnalysisResult holds all information produced by analyzing a theme file.
type AnalysisResult struct {
	Diagnostics []protocol.Diagnostic
	Palette     *color.Node
	Symbols     map[string]protocol.Range // "palette.base", "palette.highlight.low" -> definition range
	Colors      []ColorLocation
}

// ColorLocation records a resolved color at a specific source position.
type ColorLocation struct {
	Range protocol.Range
	Color color.Color
	IsRef bool // true if this is a palette reference (not a hex literal)
}

// hclPosToLSP converts an HCL position to an LSP position.
// HCL positions are 1-based; LSP positions are 0-based.
func hclPosToLSP(pos hcl.Pos) protocol.Position {
	return protocol.Position{
		Line:      uint32(pos.Line - 1),
		Character: uint32(pos.Column - 1),
	}
}

// hclRangeToLSP converts an HCL range to an LSP range.
func hclRangeToLSP(r hcl.Range) protocol.Range {
	return protocol.Range{
		Start: hclPosToLSP(r.Start),
		End:   hclPosToLSP(r.End),
	}
}

// Analyze parses HCL content from memory and produces diagnostics, a symbol table,
// and color locations. It collects ALL errors rather than short-circuiting on the first.
func Analyze(filename, content string) *AnalysisResult {
	result := &AnalysisResult{
		Symbols: make(map[string]protocol.Range),
	}

	// Parse HCL from string content
	file, diags := hclsyntax.ParseConfig([]byte(content), filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		for _, d := range diags {
			result.Diagnostics = append(result.Diagnostics, hclDiagToLSP(d))
		}
		// Cannot proceed with semantic analysis if syntax is broken
		return result
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		result.addError(hcl.Range{}, "internal error: parsed body is not *hclsyntax.Body")
		return result
	}

	// Find the palette block
	var paletteBody *hclsyntax.Body
	var themeBody *hclsyntax.Body
	var ansiBody *hclsyntax.Body
	var syntaxBody *hclsyntax.Body
	var ansiBlockRange hcl.Range

	for _, block := range body.Blocks {
		switch block.Type {
		case "palette":
			paletteBody = block.Body
		case "theme":
			themeBody = block.Body
		case "ansi":
			ansiBody = block.Body
			ansiBlockRange = block.DefRange()
		case "syntax":
			syntaxBody = block.Body
		case "meta":
			// meta is handled by gohcl in the parser; we skip it here
		}
	}

	if paletteBody == nil {
		result.addError(hcl.Range{
			Filename: filename,
			Start:    hcl.Pos{Line: 1, Column: 1},
			End:      hcl.Pos{Line: 1, Column: 1},
		}, "missing required palette block")
		return result
	}

	// Parse palette with incremental evaluation (source-ordered, self-referencing)
	palette := &color.Node{}
	result.analyzePaletteBody(paletteBody, palette, palette, "palette")
	result.Palette = palette

	// Build eval context with palette + functions
	ctx := buildAnalyzerEvalContext(palette)

	// Walk theme block
	if themeBody != nil {
		result.analyzeColorBlock(themeBody, ctx, "theme")
	}

	// Walk ansi block
	ansiColors := make(map[string]bool)
	if ansiBody != nil {
		ansiColors = result.analyzeColorBlock(ansiBody, ctx, "ansi")
	}

	// Validate ANSI completeness
	result.validateANSICompleteness(ansiColors, ansiBlockRange, filename)

	// Walk syntax block
	if syntaxBody != nil {
		result.analyzeSyntaxBody(syntaxBody, ctx, "syntax")
	}

	return result
}

// hclDiagToLSP converts an HCL diagnostic to an LSP diagnostic.
func hclDiagToLSP(d *hcl.Diagnostic) protocol.Diagnostic {
	sev := DiagError
	if d.Severity == hcl.DiagWarning {
		sev = DiagWarning
	}

	diag := protocol.Diagnostic{
		Severity: &sev,
		Message:  d.Summary,
		Source:   strPtr("pstheme"),
	}

	if d.Detail != "" {
		diag.Message = d.Summary + ": " + d.Detail
	}

	if d.Subject != nil {
		diag.Range = hclRangeToLSP(*d.Subject)
	}

	return diag
}

// addError adds an error-level diagnostic at the given range.
func (r *AnalysisResult) addError(rng hcl.Range, msg string) {
	r.Diagnostics = append(r.Diagnostics, protocol.Diagnostic{
		Range:    hclRangeToLSP(rng),
		Severity: &DiagError,
		Source:   strPtr("pstheme"),
		Message:  msg,
	})
}

// addWarning adds a warning-level diagnostic at the given range.
func (r *AnalysisResult) addWarning(rng hcl.Range, msg string) {
	r.Diagnostics = append(r.Diagnostics, protocol.Diagnostic{
		Range:    hclRangeToLSP(rng),
		Severity: &DiagWarning,
		Source:   strPtr("pstheme"),
		Message:  msg,
	})
}

func strPtr(s string) *string {
	return &s
}

// paletteItem represents an attribute or block in source order.
type paletteItem struct {
	pos   hcl.Pos
	attr  *hclsyntax.Attribute
	block *hclsyntax.Block
}

// analyzePaletteBody parses a palette block body, collecting diagnostics and building
// the symbol table and color locations. Items are processed in source order so later
// entries can reference earlier ones.
func (r *AnalysisResult) analyzePaletteBody(body *hclsyntax.Body, paletteRoot *color.Node, node *color.Node, prefix string) {
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
		// Rebuild eval context with current state of palette root
		ctx := buildAnalyzerEvalContext(paletteRoot)

		if item.attr != nil {
			attrName := item.attr.Name
			symbolName := prefix + "." + attrName

			// Record symbol for non-"color" attributes
			if attrName != "color" {
				r.Symbols[symbolName] = hclRangeToLSP(item.attr.SrcRange)
			}

			val, diags := item.attr.Expr.Value(ctx)
			if diags.HasErrors() {
				r.addError(item.attr.SrcRange, fmt.Sprintf("evaluating %s: %s", symbolName, diags.Error()))
				continue
			}

			hexStr, err := analyzerResolveColor(val)
			if err != nil {
				r.addError(item.attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, err.Error()))
				continue
			}

			c, err := color.ParseHex(hexStr)
			if err != nil {
				r.addError(item.attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, err.Error()))
				continue
			}

			// Record color location
			isRef := isReferenceExpr(item.attr.Expr)
			r.Colors = append(r.Colors, ColorLocation{
				Range: hclRangeToLSP(item.attr.Expr.Range()),
				Color: c,
				IsRef: isRef,
			})

			if attrName == "color" {
				node.Color = &c
			} else {
				if node.Children == nil {
					node.Children = make(map[string]*color.Node)
				}
				node.Children[attrName] = &color.Node{Color: &c}
			}
		} else {
			// Block: recurse
			if node.Children == nil {
				node.Children = make(map[string]*color.Node)
			}
			child := &color.Node{}
			node.Children[item.block.Type] = child
			r.analyzePaletteBody(item.block.Body, paletteRoot, child, prefix+"."+item.block.Type)
		}
	}
}

// analyzeColorBlock walks a flat color block (theme or ansi), collecting diagnostics
// and color locations. Returns a set of successfully resolved attribute names.
func (r *AnalysisResult) analyzeColorBlock(body *hclsyntax.Body, ctx *hcl.EvalContext, blockName string) map[string]bool {
	resolved := make(map[string]bool)

	for _, attr := range body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", blockName, attr.Name, diags.Error()))
			continue
		}

		hexStr, err := analyzerResolveColor(val)
		if err != nil {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", blockName, attr.Name, err.Error()))
			continue
		}

		c, err := color.ParseHex(hexStr)
		if err != nil {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", blockName, attr.Name, err.Error()))
			continue
		}

		isRef := isReferenceExpr(attr.Expr)
		r.Colors = append(r.Colors, ColorLocation{
			Range: hclRangeToLSP(attr.Expr.Range()),
			Color: c,
			IsRef: isRef,
		})

		resolved[attr.Name] = true
	}

	return resolved
}

// analyzeSyntaxBody walks a syntax block recursively, collecting diagnostics
// and color locations.
func (r *AnalysisResult) analyzeSyntaxBody(body *hclsyntax.Body, ctx *hcl.EvalContext, prefix string) {
	// Process flat attributes
	for _, attr := range body.Attributes {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", prefix, attr.Name, diags.Error()))
			continue
		}

		// In syntax, attributes can be colors or booleans (in style blocks)
		if val.Type() == cty.Bool {
			// Boolean attributes (bold, italic, underline) - skip color processing
			continue
		}

		hexStr, err := analyzerResolveColor(val)
		if err != nil {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", prefix, attr.Name, err.Error()))
			continue
		}

		c, err := color.ParseHex(hexStr)
		if err != nil {
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", prefix, attr.Name, err.Error()))
			continue
		}

		isRef := isReferenceExpr(attr.Expr)
		r.Colors = append(r.Colors, ColorLocation{
			Range: hclRangeToLSP(attr.Expr.Range()),
			Color: c,
			IsRef: isRef,
		})
	}

	// Recurse into nested blocks
	for _, block := range body.Blocks {
		r.analyzeSyntaxBody(block.Body, ctx, prefix+"."+block.Type)
	}
}

// validateANSICompleteness checks that all 16 required ANSI colors are present
// and emits warning diagnostics for any missing ones.
func (r *AnalysisResult) validateANSICompleteness(resolved map[string]bool, blockRange hcl.Range, filename string) {
	var missing []string
	for _, name := range requiredANSIColors {
		if !resolved[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		rng := blockRange
		if rng.Filename == "" {
			rng = hcl.Range{
				Filename: filename,
				Start:    hcl.Pos{Line: 1, Column: 1},
				End:      hcl.Pos{Line: 1, Column: 1},
			}
		}
		r.addWarning(rng, fmt.Sprintf("ANSI block missing colors: %s", strings.Join(missing, ", ")))
	}
}

// analyzerResolveColor extracts a color hex string from a cty.Value.
// If the value is a string, return it directly.
// If the value is an object, extract the "color" key.
func analyzerResolveColor(val cty.Value) (string, error) {
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

// isReferenceExpr returns true if the expression is a scope traversal
// (e.g. palette.base) rather than a literal value.
func isReferenceExpr(expr hclsyntax.Expression) bool {
	switch expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		return true
	case *hclsyntax.RelativeTraversalExpr:
		return true
	default:
		return false
	}
}

// nodeToCtyAnalyzer converts a color.Node to a cty.Value for HCL evaluation context.
// This is a reimplementation for the analyzer to avoid coupling to the parser package.
func nodeToCtyAnalyzer(node *color.Node) cty.Value {
	if node.Children == nil {
		if node.Color != nil {
			return cty.StringVal(node.Color.Hex())
		}
		return cty.EmptyObjectVal
	}

	vals := make(map[string]cty.Value, len(node.Children)+1)

	if node.Color != nil {
		vals["color"] = cty.StringVal(node.Color.Hex())
	}

	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals[k] = nodeToCtyAnalyzer(node.Children[k])
	}

	return cty.ObjectVal(vals)
}

// makeBrightenFuncAnalyzer creates an HCL function that brightens a color.
func makeBrightenFuncAnalyzer() function.Function {
	return function.New(&function.Spec{
		Description: "Brightens a color by the given percentage (-1.0 to 1.0)",
		Params: []function.Parameter{
			{Name: "color", Type: cty.String},
			{Name: "percentage", Type: cty.Number},
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

// makeDarkenFuncAnalyzer creates an HCL function that darkens a color.
func makeDarkenFuncAnalyzer() function.Function {
	return function.New(&function.Spec{
		Description: "Darkens a color by the given percentage (0.0 to 1.0)",
		Params: []function.Parameter{
			{Name: "color", Type: cty.String},
			{Name: "percentage", Type: cty.Number},
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

// buildAnalyzerEvalContext creates an HCL evaluation context with palette variables
// and brighten/darken functions.
func buildAnalyzerEvalContext(palette *color.Node) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": nodeToCtyAnalyzer(palette),
		},
		Functions: map[string]function.Function{
			"brighten": makeBrightenFuncAnalyzer(),
			"darken":   makeDarkenFuncAnalyzer(),
		},
	}
}
