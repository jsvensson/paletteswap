# Universal Block Reference System - Implementation Guide

## Overview

This document provides step-by-step implementation details for transforming the palette-only reference system into a universal block reference system.

## Current Architecture

### Block Processing Functions
1. `analyzePaletteBody` - Processes palette with self-referencing and nesting
2. `analyzeColorBlock` - Processes theme/ansi (flat, no self-referencing)
3. `analyzeSyntaxBody` - Processes syntax (nested, no self-referencing)

### Symbol Table
- Keys: `"palette.base"`, `"palette.highlight.low"`
- Stores definition locations for go-to-definition

### Eval Context
- Only contains `palette` variable
- Functions: `brighten`, `darken`

## Target Architecture

### Unified Block Processing
Single `analyzeBlock` function handles all blocks based on `BlockType` configuration.

### Symbol Table
- Keys: `"palette"`, `"palette.base"`, `"theme"`, `"theme.background"`, `"syntax"`, `"syntax.keyword"`, `"ansi"`, `"ansi.red"`
- All blocks and their properties stored

### Eval Context
- Variables: `palette`, `theme`, `syntax`, `ansi`
- Functions: `brighten`, `darken`
- All blocks available for cross-referencing

## Implementation Steps

### Step 1: Add BlockContext and blockItem Types

Add to `analyzer.go` after `paletteItem`:

```go
// blockItem represents an attribute or block in source order.
// Replaces paletteItem for universal block processing.
type blockItem struct {
	pos   hcl.Pos
	attr  *hclsyntax.Attribute
	block *hclsyntax.Block
}

// BlockContext holds the state for a block being analyzed
type BlockContext struct {
	Name       string
	BlockType  BlockType
	Node       *color.Node  // For building color tree
	Symbols    map[string]protocol.Range
	Items      []blockItem
}
```

### Step 2: Create analyzeBlock Function

Replace `analyzePaletteBody`, `analyzeColorBlock`, and `analyzeSyntaxBody` with unified function:

```go
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
```

### Step 3: Create Helper Functions

```go
// isValidANSIName checks if a name is in the list of valid ANSI colors
func isValidANSIName(name string) bool {
    for _, valid := range requiredANSIColors {
        if name == valid {
            return true
        }
    }
    return false
}

// hasCircularReference checks if an expression references something not yet defined
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
        newCtx.Variables[blockName] = node.ToCty()
    }
    
    return newCtx
}
```

### Step 4: Create processBlockAttribute

```go
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
        r.addError(attr.SrcRange, fmt.Sprintf("%s: %s", symbolName, diags.Error()))
        return
    }
    
    // Handle boolean attributes (bold, italic, underline in syntax)
    if val.Type() == cty.Bool {
        ctx.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)
        r.Symbols[symbolName] = hclRangeToLSP(attr.SrcRange)
        resolved[attr.Name] = true
        return
    }
    
    hexStr, err := analyzerResolveColor(val)
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
    
    // Update node tree
    if ctx.Node.Children == nil {
        ctx.Node.Children = make(map[string]*color.Node)
    }
    ctx.Node.Children[attr.Name] = &color.Node{Color: &c}
    
    resolved[attr.Name] = true
}
```

### Step 5: Create processBlockNestedBlock

```go
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
```

### Step 6: Update Analyze Function

Replace the block processing in `Analyze()`:

```go
func Analyze(filename, content string) *AnalysisResult {
    result := &AnalysisResult{
        Symbols:     make(map[string]protocol.Range),
        Diagnostics: []protocol.Diagnostic{},
    }

    // Parse HCL
    file, diags := hclsyntax.ParseConfig([]byte(content), filename, hcl.Pos{Line: 1, Column: 1})
    
    for _, d := range diags {
        if lspDiag := hclDiagToLSP(d); lspDiag != nil {
            result.Diagnostics = append(result.Diagnostics, *lspDiag)
        }
    }
    
    if file == nil || file.Body == nil {
        return result
    }
    
    body, ok := file.Body.(*hclsyntax.Body)
    if !ok {
        result.addError(hcl.Range{}, "internal error: parsed body is not *hclsyntax.Body")
        return result
    }
    
    // Track blocks for processing
    var blockBodies = make(map[string]*hclsyntax.Body)
    var blockRanges = make(map[string]hcl.Range)
    
    // First pass: collect all blocks and store their locations
    for _, block := range body.Blocks {
        if blockType, exists := BlockTypes[block.Type]; exists {
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
    
    // Build initial eval context (empty, will be populated as we process)
    ctx := &hcl.EvalContext{
        Variables: make(map[string]cty.Value),
        Functions: make(map[string]function.Function),
    }
    ctx.Functions["brighten"] = brightenFunc
    ctx.Functions["darken"] = darkenFunc
    
    // Process palette first (required and may be referenced by others)
    if paletteBody, ok := blockBodies["palette"]; ok {
        palette := &color.Node{}
        result.analyzeBlock(paletteBody, BlockTypes["palette"], ctx, "palette")
        result.Palette = palette
        ctx.Variables["palette"] = palette.ToCty()
    }
    
    // Process theme (self-referencing, can reference palette)
    if themeBody, ok := blockBodies["theme"]; ok {
        themeNode, _ := result.analyzeBlock(themeBody, BlockTypes["theme"], ctx, "theme")
        ctx.Variables["theme"] = themeNode.ToCty()
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
```

### Step 7: Update Definition.go

Replace `paletteRefAtCursor` with `blockRefAtCursor`:

```go
// blockRefAtCursor extracts the block reference path up to the cursor position.
// Works with any block: palette, theme, ansi, syntax
func blockRefAtCursor(line string, character uint32) string {
    col := int(character)
    if col >= len(line) {
        return ""
    }

    // Find the end of the current word
    end := col
    for end < len(line) && isIdentChar(line[end]) {
        end++
    }

    // Find the start of the current word
    start := col
    for start > 0 && isIdentChar(line[start-1]) {
        start--
    }

    word := line[start:end]
    
    // Check if it's a valid block reference
    parts := strings.Split(word, ".")
    if len(parts) == 0 {
        return ""
    }
    
    // Check if first part is a valid block name
    if _, exists := BlockTypes[parts[0]]; !exists {
        return ""
    }
    
    // If cursor is on just the block name, check if followed by dot
    if len(parts) == 1 && word == parts[0] {
        if end < len(line) && line[end] == '.' {
            return parts[0]
        }
        return ""
    }
    
    // Calculate cursor position within word and return path up to cursor
    cursorInWord := col - start
    var resultParts []string
    currentPos := 0
    
    for _, part := range parts {
        partEnd := currentPos + len(part)
        if currentPos <= cursorInWord {
            resultParts = append(resultParts, part)
        }
        currentPos = partEnd + 1 // +1 for dot
    }
    
    return strings.Join(resultParts, ".")
}
```

Update `definition` function to use `blockRefAtCursor`.

### Step 8: Update Semantic.go

Update `extractTokensFromTraversal`:

```go
func extractTokensFromTraversal(expr *hclsyntax.ScopeTraversalExpr, tokens []SemanticToken) []SemanticToken {
    if len(expr.Traversal) == 0 {
        return tokens
    }

    first, ok := expr.Traversal[0].(hcl.TraverseRoot)
    if !ok {
        return tokens
    }
    
    // Check if it's a referenceable block
    if _, exists := BlockTypes[first.Name]; !exists {
        return tokens
    }

    // Tokenize block name as namespace
    tokens = append(tokens, SemanticToken{
        Line:      uint32(first.SrcRange.Start.Line - 1),
        StartChar: uint32(first.SrcRange.Start.Column - 1),
        Length:    uint32(len(first.Name)),
        Type:      tokenTypeIndices["namespace"],
        Modifiers: 0,
    })

    // Tokenize each subsequent segment as property
    for i := 1; i < len(expr.Traversal); i++ {
        switch seg := expr.Traversal[i].(type) {
        case hcl.TraverseAttr:
            tokens = append(tokens, SemanticToken{
                Line:      uint32(seg.SrcRange.Start.Line - 1),
                StartChar: uint32(seg.SrcRange.Start.Column),
                Length:    uint32(len(seg.Name)),
                Type:      tokenTypeIndices["property"],
                Modifiers: 0,
            })
        case hcl.TraverseIndex:
            // Handle index access if needed
        }
    }

    return tokens
}
```

## Testing Checklist

- [ ] Cross-block references work (syntax.keyword = ansi.red)
- [ ] Self-referencing works (syntax.operator = syntax.keyword)
- [ ] Nested theme works (theme.panel.background = palette.surface)
- [ ] ANSI strict validation works (rejects invalid color names)
- [ ] Circular reference detection works
- [ ] Go-to-definition works for all blocks
- [ ] Semantic tokens work for all blocks
- [ ] Completion works for all blocks
- [ ] All existing tests pass
- [ ] New tests added for all features

## Migration Notes

- Symbol table format remains compatible (still uses dot notation)
- Template functions should work without changes (they use dot paths)
- Existing themes should continue to work
- Only new features (cross-block refs, self-refs) require theme updates
