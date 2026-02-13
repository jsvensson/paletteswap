package lsp

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
	"github.com/zclconf/go-cty/cty/function"
)

var (
	DiagError   = protocol.DiagnosticSeverityError
	DiagWarning = protocol.DiagnosticSeverityWarning
	DiagInfo    = protocol.DiagnosticSeverityInformation
)

// BlockType defines the behavior of each top-level block
type BlockType struct {
	Name            string
	SupportsNesting bool     // theme, syntax, palette = true; ansi = false
	SelfReferencing bool     // Can reference earlier items in same block
	StrictNames     []string // For ANSI: only these names allowed
}

// BlockTypes defines the configuration for each referenceable block
var BlockTypes = map[string]BlockType{
	"palette": {
		Name:            "palette",
		SupportsNesting: true,
		SelfReferencing: true,
	},
	"theme": {
		Name:            "theme",
		SupportsNesting: true,
		SelfReferencing: true,
	},
	"syntax": {
		Name:            "syntax",
		SupportsNesting: true,
		SelfReferencing: true,
	},
	"ansi": {
		Name:            "ansi",
		SupportsNesting: false,
		SelfReferencing: false,
		StrictNames:     theme.RequiredANSIColors,
	},
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
		Symbols:     make(map[string]protocol.Range),
		Diagnostics: []protocol.Diagnostic{}, // Initialize to empty slice, not nil
	}

	// Parse HCL from string content
	file, diags := hclsyntax.ParseConfig([]byte(content), filename, hcl.Pos{Line: 1, Column: 1})

	// Convert HCL diagnostics, filtering out unhelpful ones during editing
	for _, d := range diags {
		if lspDiag := hclDiagToLSP(d); lspDiag != nil {
			result.Diagnostics = append(result.Diagnostics, *lspDiag)
		}
	}

	// Only return early if we truly can't proceed (no file or body)
	if file == nil || file.Body == nil {
		return result
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		result.addError(hcl.Range{}, "internal error: parsed body is not *hclsyntax.Body")
		return result
	}

	// Track blocks for processing
	blockBodies := make(map[string]*hclsyntax.Body)
	blockRanges := make(map[string]hcl.Range)

	// First pass: collect all blocks and store their locations
	for _, block := range body.Blocks {
		if _, exists := BlockTypes[block.Type]; exists {
			blockBodies[block.Type] = block.Body
			blockRanges[block.Type] = block.DefRange()
			// Store block location in symbols
			result.Symbols[block.Type] = hclRangeToLSP(block.DefRange())
		}
	}

	// Check for required palette block
	if _, hasPalette := blockBodies["palette"]; !hasPalette {
		result.addError(hcl.Range{
			Filename: filename,
			Start:    hcl.Pos{Line: 1, Column: 1},
			End:      hcl.Pos{Line: 1, Column: 1},
		}, "missing required palette block")
		return result
	}

	// Build initial eval context with functions
	ctx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: map[string]function.Function{
			"brighten": theme.MakeBrightenFunc(),
			"darken":   theme.MakeDarkenFunc(),
		},
	}

	// Process palette first (required and may be referenced by others)
	if paletteBody, ok := blockBodies["palette"]; ok {
		palette, _ := result.analyzeBlock(paletteBody, BlockTypes["palette"], ctx, "palette")
		result.Palette = palette
		ctx.Variables["palette"] = theme.NodeToCty(palette)
	}

	// Process theme (self-referencing, can reference palette)
	if themeBody, ok := blockBodies["theme"]; ok {
		themeNode, _ := result.analyzeBlock(themeBody, BlockTypes["theme"], ctx, "theme")
		ctx.Variables["theme"] = theme.NodeToCty(themeNode)
	}

	// Process ansi (strict names, can reference palette/theme)
	if ansiBody, ok := blockBodies["ansi"]; ok {
		_, ansiResolved := result.analyzeBlock(ansiBody, BlockTypes["ansi"], ctx, "ansi")
		result.validateANSICompleteness(ansiResolved, blockRanges["ansi"], filename)
	}

	// Process syntax (self-referencing, can reference all others)
	if syntaxBody, ok := blockBodies["syntax"]; ok {
		_, _ = result.analyzeBlock(syntaxBody, BlockTypes["syntax"], ctx, "syntax")
	}

	return result
}

// hclDiagToLSP converts an HCL diagnostic to an LSP diagnostic.
// Returns nil if the diagnostic should be filtered out (e.g., unhelpful editing errors).
func hclDiagToLSP(d *hcl.Diagnostic) *protocol.Diagnostic {
	// Filter out "Invalid attribute name" errors during editing
	// These occur when user types "palette." and hasn't typed the attribute yet
	if d.Summary == "Invalid attribute name" && strings.Contains(d.Detail, "required after a dot") {
		return nil
	}

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

	return &diag
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
		ctx := theme.BuildEvalContext(paletteRoot)

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

			hexStr, err := theme.ResolveColor(val)
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
			// Filter out "Invalid attribute name" errors during editing
			// These occur when user types "palette." and hasn't typed the attribute yet
			errStr := diags.Error()
			if strings.Contains(errStr, "Invalid attribute name") {
				continue
			}
			r.addError(attr.SrcRange, fmt.Sprintf("%s.%s: %s", blockName, attr.Name, errStr))
			continue
		}

		hexStr, err := theme.ResolveColor(val)
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

		hexStr, err := theme.ResolveColor(val)
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
	for _, name := range theme.RequiredANSIColors {
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

// blockItem represents an attribute or block in source order.
type blockItem struct {
	pos   hcl.Pos
	attr  *hclsyntax.Attribute
	block *hclsyntax.Block
}

// BlockContext holds the state for a block being analyzed
type BlockContext struct {
	Name      string
	BlockType BlockType
	Node      *color.Node // For building color tree
	Symbols   map[string]protocol.Range
	Items     []blockItem
}

// isValidANSIName checks if a name is in the list of valid ANSI colors
func isValidANSIName(name string) bool {
	for _, valid := range theme.RequiredANSIColors {
		if name == valid {
			return true
		}
	}
	return false
}

// hasCircularReference checks if an expression references something not yet defined
// within the current block being analyzed
func (r *AnalysisResult) hasCircularReference(expr hclsyntax.Expression, currentPrefix string) bool {
	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		if len(e.Traversal) > 0 {
			if root, ok := e.Traversal[0].(hcl.TraverseRoot); ok {
				var parts []string
				parts = append(parts, root.Name)
				for _, t := range e.Traversal[1:] {
					if attr, ok := t.(hcl.TraverseAttr); ok {
						parts = append(parts, attr.Name)
					}
				}
				refPath := strings.Join(parts, ".")

				// Check if referencing current block with path not yet defined
				if strings.HasPrefix(refPath, currentPrefix+".") {
					if _, exists := r.Symbols[refPath]; !exists {
						return true
					}
				}
			}
		}
	case *hclsyntax.FunctionCallExpr:
		for _, arg := range e.Args {
			if r.hasCircularReference(arg, currentPrefix) {
				return true
			}
		}
	}
	return false
}

// analyzeBlock processes any block type with unified logic
func (r *AnalysisResult) analyzeBlock(body *hclsyntax.Body, blockType BlockType,
	parentCtx *hcl.EvalContext, prefix string) (*color.Node, map[string]bool) {

	ctx := &BlockContext{
		Name:      blockType.Name,
		BlockType: blockType,
		Node:      &color.Node{},
		Symbols:   make(map[string]protocol.Range),
		Items:     []blockItem{},
	}

	// Collect items
	for _, attr := range body.Attributes {
		// Validate ANSI names if strict
		if blockType.StrictNames != nil {
			if !isValidANSIName(attr.Name) {
				r.addError(attr.SrcRange,
					fmt.Sprintf("ansi.%s is not a valid ANSI color name", attr.Name))
				continue
			}
		}
		ctx.Items = append(ctx.Items, blockItem{pos: attr.SrcRange.Start, attr: attr})
	}

	for _, block := range body.Blocks {
		if !blockType.SupportsNesting {
			r.addError(block.DefRange(),
				fmt.Sprintf("%s block does not support nesting", blockType.Name))
			continue
		}
		ctx.Items = append(ctx.Items, blockItem{pos: block.DefRange().Start, block: block})
	}

	// Sort by source position for self-referencing blocks
	if blockType.SelfReferencing {
		sort.Slice(ctx.Items, func(i, j int) bool {
			if ctx.Items[i].pos.Line != ctx.Items[j].pos.Line {
				return ctx.Items[i].pos.Line < ctx.Items[j].pos.Line
			}
			return ctx.Items[i].pos.Column < ctx.Items[j].pos.Column
		})
	}

	// Process items
	resolved := make(map[string]bool)
	currentCtx := parentCtx

	for _, item := range ctx.Items {
		// Rebuild context after each item for self-referencing blocks
		if blockType.SelfReferencing {
			currentCtx = r.buildBlockEvalContext(currentCtx, prefix, ctx.Node)
		}

		if item.attr != nil {
			r.processBlockAttribute(item.attr, ctx, currentCtx, prefix, resolved)
		} else {
			r.processBlockNestedBlock(item.block, ctx, currentCtx, prefix, resolved)
		}
	}

	return ctx.Node, resolved
}

// processBlockAttribute processes a single attribute in a block
func (r *AnalysisResult) processBlockAttribute(attr *hclsyntax.Attribute,
	ctx *BlockContext, evalCtx *hcl.EvalContext, prefix string, resolved map[string]bool) {

	symbolName := prefix + "." + attr.Name

	// Check for circular references
	if ctx.BlockType.SelfReferencing && r.hasCircularReference(attr.Expr, prefix) {
		r.addError(attr.SrcRange, fmt.Sprintf("circular reference detected in %s", symbolName))
		return
	}

	val, diags := attr.Expr.Value(evalCtx)
	if diags.HasErrors() {
		errStr := diags.Error()
		// Filter out "Invalid attribute name" errors during editing
		// These occur when user types "palette." and hasn't typed the attribute yet
		if strings.Contains(errStr, "Invalid attribute name") {
			return
		}
		r.addError(attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, errStr))
		return
	}

	// Handle boolean attributes (bold, italic, underline in syntax)
	if val.Type() == cty.Bool {
		ctx.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)
		r.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)
		resolved[attr.Name] = true
		return
	}

	hexStr, err := theme.ResolveColor(val)
	if err != nil {
		r.addError(attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, err.Error()))
		return
	}

	c, err := color.ParseHex(hexStr)
	if err != nil {
		r.addError(attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, err.Error()))
		return
	}

	// Record color location
	isRef := isReferenceExpr(attr.Expr)
	r.Colors = append(r.Colors, ColorLocation{
		Range: hclRangeToLSP(attr.Expr.Range()),
		Color: c,
		IsRef: isRef,
	})

	// Store symbol
	ctx.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)
	r.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)

	// Update node tree — "color" is a reserved keyword that sets the node's
	// own color rather than creating a child entry.
	if attr.Name == "color" && ctx.BlockType.SupportsNesting {
		ctx.Node.Color = &c
	} else {
		if ctx.Node.Children == nil {
			ctx.Node.Children = make(map[string]*color.Node)
		}
		ctx.Node.Children[attr.Name] = &color.Node{Color: &c}
	}

	resolved[attr.Name] = true
}

// processBlockNestedBlock processes a nested block
func (r *AnalysisResult) processBlockNestedBlock(block *hclsyntax.Block,
	ctx *BlockContext, evalCtx *hcl.EvalContext, prefix string, resolved map[string]bool) {

	childPrefix := prefix + "." + block.Type

	// Store nested block symbol
	ctx.Symbols[childPrefix] = hclRangeToLSP(block.DefRange())
	r.Symbols[childPrefix] = hclRangeToLSP(block.DefRange())

	// Recursively analyze nested block with same block type
	childNode, _ := r.analyzeBlock(block.Body, ctx.BlockType, evalCtx, childPrefix)

	// Add to parent node
	if ctx.Node.Children == nil {
		ctx.Node.Children = make(map[string]*color.Node)
	}
	ctx.Node.Children[block.Type] = childNode
}

// buildBlockEvalContext rebuilds eval context with current block state
func (r *AnalysisResult) buildBlockEvalContext(parentCtx *hcl.EvalContext,
	blockName string, node *color.Node) *hcl.EvalContext {

	// Copy parent context
	newCtx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
		Functions: parentCtx.Functions,
	}
	for k, v := range parentCtx.Variables {
		newCtx.Variables[k] = v
	}

	// Update this block's variable
	if node != nil {
		newCtx.Variables[blockName] = theme.NodeToCty(node)
	}

	return newCtx
}
