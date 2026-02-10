package lsp

import (
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Semantic token types we'll use (indices 0-7)
var semanticTokenTypes = []string{
	"keyword",   // 0: block names (meta, palette, theme, ansi, syntax)
	"property",  // 1: attribute names
	"variable",  // 2: palette references (kept for non-palette refs if needed)
	"namespace", // 3: the "palette" namespace identifier
	"string",    // 4: hex color literals
	"function",  // 5: brighten(), darken()
	"number",    // 6: numeric literals
	"comment",   // 7: comments
}

// Semantic token modifiers (bit flags)
var semanticTokenModifiers = []string{
	"declaration", // bit 0: defining a new symbol
}

// tokenTypeIndices maps type names to their indices for fast lookup
var tokenTypeIndices map[string]uint32

func init() {
	tokenTypeIndices = make(map[string]uint32, len(semanticTokenTypes))
	for i, t := range semanticTokenTypes {
		tokenTypeIndices[t] = uint32(i)
	}
}

// SemanticToken represents a single token with its metadata
type SemanticToken struct {
	Line      uint32 // 0-based line number
	StartChar uint32 // 0-based character offset
	Length    uint32
	Type      uint32 // index into semanticTokenTypes
	Modifiers uint32 // bit flags
}

// encodeTokens converts tokens to LSP format (5 integers per token)
// Uses delta encoding for line numbers and character positions
func encodeTokens(tokens []SemanticToken) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}

	// Sort tokens by position
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Line != tokens[j].Line {
			return tokens[i].Line < tokens[j].Line
		}
		return tokens[i].StartChar < tokens[j].StartChar
	})

	data := make([]uint32, 0, len(tokens)*5)

	var prevLine uint32 = 0
	var prevChar uint32 = 0

	for _, tok := range tokens {
		deltaLine := tok.Line - prevLine
		var deltaStart uint32
		if deltaLine == 0 {
			deltaStart = tok.StartChar - prevChar
		} else {
			deltaStart = tok.StartChar
		}

		data = append(data,
			deltaLine,
			deltaStart,
			tok.Length,
			tok.Type,
			tok.Modifiers,
		)

		prevLine = tok.Line
		prevChar = tok.StartChar
	}

	return data
}

// semanticTokensFull generates semantic tokens for the entire document content
func semanticTokensFull(content string) []uint32 {
	file, diags := hclsyntax.ParseConfig([]byte(content), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		// Return empty tokens if parsing fails
		return []uint32{}
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return []uint32{}
	}

	var tokens []SemanticToken
	tokens = extractTokensFromBody(body, tokens)

	return encodeTokens(tokens)
}

// extractTokensFromBody extracts tokens from an HCL body
func extractTokensFromBody(body *hclsyntax.Body, tokens []SemanticToken) []SemanticToken {
	// Extract block type tokens
	for _, block := range body.Blocks {
		tokens = append(tokens, SemanticToken{
			Line:      uint32(block.DefRange().Start.Line - 1),
			StartChar: uint32(block.DefRange().Start.Column - 1),
			Length:    uint32(len(block.Type)),
			Type:      tokenTypeIndices["keyword"],
			Modifiers: 0,
		})

		// Recurse into block body
		tokens = extractTokensFromBody(block.Body, tokens)
	}

	// Extract attribute tokens
	for name, attr := range body.Attributes {
		// Attribute name (with declaration modifier)
		tokens = append(tokens, SemanticToken{
			Line:      uint32(attr.SrcRange.Start.Line - 1),
			StartChar: uint32(attr.SrcRange.Start.Column - 1),
			Length:    uint32(len(name)),
			Type:      tokenTypeIndices["property"],
			Modifiers: 1, // declaration bit
		})

		// Extract tokens from the expression
		tokens = extractTokensFromExpr(attr.Expr, tokens)
	}

	return tokens
}

// extractTokensFromExpr extracts tokens from an HCL expression
func extractTokensFromExpr(expr hclsyntax.Expression, tokens []SemanticToken) []SemanticToken {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		tokens = extractTokensFromLiteral(e, tokens)
	case *hclsyntax.ScopeTraversalExpr:
		tokens = extractTokensFromTraversal(e, tokens)
	case *hclsyntax.FunctionCallExpr:
		tokens = extractTokensFromFunctionCall(e, tokens)
	case *hclsyntax.RelativeTraversalExpr:
		tokens = extractTokensFromRelativeTraversal(e, tokens)
	}
	return tokens
}

// extractTokensFromLiteral handles string and number literals
func extractTokensFromLiteral(expr *hclsyntax.LiteralValueExpr, tokens []SemanticToken) []SemanticToken {
	val := expr.Val
	if val.Type().FriendlyName() == "string" {
		str := val.AsString()
		// Check if it's a hex color
		if len(str) == 7 && str[0] == '#' {
			tokens = append(tokens, SemanticToken{
				Line:      uint32(expr.SrcRange.Start.Line - 1),
				StartChar: uint32(expr.SrcRange.Start.Column - 1),
				Length:    uint32(len(str)),
				Type:      tokenTypeIndices["string"],
				Modifiers: 0,
			})
		}
	} else if val.Type().FriendlyName() == "number" {
		tokens = append(tokens, SemanticToken{
			Line:      uint32(expr.SrcRange.Start.Line - 1),
			StartChar: uint32(expr.SrcRange.Start.Column - 1),
			Length:    uint32(expr.SrcRange.End.Column - expr.SrcRange.Start.Column),
			Type:      tokenTypeIndices["number"],
			Modifiers: 0,
		})
	}
	return tokens
}

// extractTokensFromTraversal handles any block reference like palette.base or ansi.red
func extractTokensFromTraversal(expr *hclsyntax.ScopeTraversalExpr, tokens []SemanticToken) []SemanticToken {
	if len(expr.Traversal) == 0 {
		return tokens
	}

	// Check if first segment is a valid block name
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
				StartChar: uint32(seg.SrcRange.Start.Column - 1),
				Length:    uint32(len(seg.Name)),
				Type:      tokenTypeIndices["property"],
				Modifiers: 0,
			})
		case hcl.TraverseIndex:
			// Handle index access like palette.colors[0] if needed
			// For now, skip or handle as needed
		}
	}

	return tokens
}

// extractTokensFromFunctionCall handles function calls like brighten()
func extractTokensFromFunctionCall(expr *hclsyntax.FunctionCallExpr, tokens []SemanticToken) []SemanticToken {
	// Tokenize the function name
	tokens = append(tokens, SemanticToken{
		Line:      uint32(expr.NameRange.Start.Line - 1),
		StartChar: uint32(expr.NameRange.Start.Column - 1),
		Length:    uint32(len(expr.Name)),
		Type:      tokenTypeIndices["function"],
		Modifiers: 0,
	})

	// Recurse into arguments
	for _, arg := range expr.Args {
		tokens = extractTokensFromExpr(arg, tokens)
	}

	return tokens
}

// extractTokensFromRelativeTraversal handles relative traversals
func extractTokensFromRelativeTraversal(expr *hclsyntax.RelativeTraversalExpr, tokens []SemanticToken) []SemanticToken {
	// For now, just recurse into the source
	return extractTokensFromExpr(expr.Source, tokens)
}
