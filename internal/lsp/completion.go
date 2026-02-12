package lsp

import (
	"strings"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/theme"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// splitLines splits content into lines, preserving empty trailing lines.
func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

// blockContext represents the kind of block the cursor is in.
type blockContext int

const (
	contextRoot    blockContext = iota
	contextMeta                 // inside meta {}
	contextPalette              // inside palette {}
	contextTheme                // inside theme {}
	contextAnsi                 // inside ansi {}
	contextSyntax               // inside syntax {} (top level)
	contextStyle                // inside a sub-block of syntax {} (style block)
)

// styleAttributes are the valid attributes inside a syntax style block.
var styleAttributes = []string{"color", "bold", "italic", "underline"}

// topLevelBlocks are the valid top-level block names.
var topLevelBlocks = []string{"meta", "palette", "theme", "syntax", "ansi"}

// complete produces completion items given an analysis result, document content,
// and cursor position. This is the core logic, decoupled from the LSP protocol
// handler for testability.
func complete(result *AnalysisResult, content string, pos protocol.Position) []protocol.CompletionItem {
	lines := splitLines(content)
	if int(pos.Line) >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	charPos := min(int(pos.Character), len(line))
	textBeforeCursor := line[:charPos]

	// Check for palette path completion: look for "palette." or "palette.xxx."
	if paletteItems := tryPaletteCompletion(result, textBeforeCursor); paletteItems != nil {
		return paletteItems
	}

	// Check for value position (after "=") — offer functions and palette
	if isValuePosition(textBeforeCursor) {
		return valueCompletions()
	}

	// Determine which block the cursor is in by scanning backwards
	ctx := determineBlockContext(lines, int(pos.Line))

	switch ctx {
	case contextAnsi:
		return ansiCompletions(lines, int(pos.Line))
	case contextStyle:
		return styleCompletions(lines, int(pos.Line))
	case contextRoot:
		return topLevelCompletions()
	}

	return nil
}

// tryPaletteCompletion checks if the text before the cursor ends with a palette
// path prefix (e.g., "palette." or "palette.highlight.") and returns completion
// items for the children at that node in the palette tree.
func tryPaletteCompletion(result *AnalysisResult, textBeforeCursor string) []protocol.CompletionItem {
	if result == nil || result.Palette == nil {
		return nil
	}

	// Find the last occurrence of "palette." in the text before cursor
	idx := strings.LastIndex(textBeforeCursor, "palette.")
	if idx == -1 {
		return nil
	}

	// Extract the path after "palette."
	pathStr := textBeforeCursor[idx+len("palette."):]

	// Walk the palette tree based on the path segments.
	// - "palette."              -> children of root (segments = nil)
	// - "palette.highlight."    -> children of "highlight" node
	// - "palette.high"          -> children of root (client filters partial match)
	// - "palette.highlight.lo"  -> children of "highlight" (client filters "lo")
	var segments []string
	if pathStr == "" {
		segments = nil
	} else if before, ok := strings.CutSuffix(pathStr, "."); ok {
		trimmed := before
		segments = strings.Split(trimmed, ".")
	} else if strings.Contains(pathStr, ".") {
		parts := strings.Split(pathStr, ".")
		segments = parts[:len(parts)-1]
	} else {
		segments = nil
	}

	// Walk the palette tree to the target node
	node := result.Palette
	for _, seg := range segments {
		if node.Children == nil {
			return nil
		}
		child, ok := node.Children[seg]
		if !ok {
			return nil
		}
		node = child
	}

	if node.Children == nil {
		return nil
	}

	return nodeChildrenToCompletionItems(node)
}

// nodeChildrenToCompletionItems converts a node's children into completion items.
func nodeChildrenToCompletionItems(node *color.Node) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	kind := protocol.CompletionItemKindColor

	for name, child := range node.Children {
		item := protocol.CompletionItem{
			Label: name,
			Kind:  &kind,
		}

		// If the child has a direct color, show it in Detail
		if child.Color != nil {
			hex := child.Color.Hex()
			item.Detail = &hex
		} else if child.Children != nil {
			// It's a group/namespace — still offer it but with a different detail
			groupKind := protocol.CompletionItemKindModule
			item.Kind = &groupKind
			detail := "color group"
			item.Detail = &detail
		}

		items = append(items, item)
	}

	return items
}

// isValuePosition returns true if the text before the cursor indicates we are
// at a value position (after an "=" sign with nothing meaningful following it).
func isValuePosition(textBeforeCursor string) bool {
	trimmed := strings.TrimSpace(textBeforeCursor)
	eqIdx := strings.LastIndex(trimmed, "=")
	if eqIdx == -1 {
		return false
	}
	afterEq := strings.TrimSpace(trimmed[eqIdx+1:])
	return afterEq == ""
}

// valueCompletions returns completion items for a value position, including
// function snippets and a palette reference trigger.
func valueCompletions() []protocol.CompletionItem {
	snippetFormat := protocol.InsertTextFormatSnippet

	brightenSnippet := "brighten(${1:color}, ${2:0.1})"
	darkenSnippet := "darken(${1:color}, ${2:0.1})"
	paletteSnippet := "palette."

	return []protocol.CompletionItem{
		{
			Label:            "brighten",
			Kind:             completionKindPtr(protocol.CompletionItemKindFunction),
			Detail:           strPtr("brighten(color, percentage)"),
			InsertText:       &brightenSnippet,
			InsertTextFormat: &snippetFormat,
		},
		{
			Label:            "darken",
			Kind:             completionKindPtr(protocol.CompletionItemKindFunction),
			Detail:           strPtr("darken(color, percentage)"),
			InsertText:       &darkenSnippet,
			InsertTextFormat: &snippetFormat,
		},
		{
			Label:      "palette",
			Kind:       completionKindPtr(protocol.CompletionItemKindVariable),
			Detail:     strPtr("palette reference"),
			InsertText: &paletteSnippet,
		},
	}
}

// determineBlockContext scans from the top of the file down to the cursor line
// to determine which block the cursor is in, using brace nesting.
func determineBlockContext(lines []string, cursorLine int) blockContext {
	type blockInfo struct {
		name string
	}

	var stack []blockInfo

	for i := 0; i <= cursorLine; i++ {
		line := strings.TrimSpace(lines[i])

		opens := strings.Count(line, "{")
		closes := strings.Count(line, "}")

		// Process opening braces: extract the block name (first word on the line)
		if opens > 0 {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				name := parts[0]
				for range opens {
					stack = append(stack, blockInfo{name: name})
				}
			}
		}

		// Process closing braces
		if closes > 0 {
			for range closes {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	if len(stack) == 0 {
		return contextRoot
	}

	current := stack[len(stack)-1]

	switch current.name {
	case "meta":
		return contextMeta
	case "palette":
		return contextPalette
	case "theme":
		return contextTheme
	case "ansi":
		return contextAnsi
	case "syntax":
		return contextSyntax
	default:
		// If the parent is "syntax", we're in a style sub-block
		if len(stack) >= 2 {
			parent := stack[len(stack)-2]
			if parent.name == "syntax" {
				return contextStyle
			}
		}
		return contextRoot
	}
}

// ansiCompletions returns ANSI color name completions, excluding names that are
// already defined in the ansi block surrounding the cursor.
func ansiCompletions(lines []string, cursorLine int) []protocol.CompletionItem {
	defined := findDefinedAttributes(lines, cursorLine)
	kind := protocol.CompletionItemKindConstant

	var items []protocol.CompletionItem
	for _, name := range theme.RequiredANSIColors {
		if !defined[name] {
			items = append(items, protocol.CompletionItem{
				Label: name,
				Kind:  &kind,
			})
		}
	}

	return items
}

// styleCompletions returns style attribute completions, excluding attributes
// already defined in the current style block.
func styleCompletions(lines []string, cursorLine int) []protocol.CompletionItem {
	defined := findDefinedAttributes(lines, cursorLine)
	kind := protocol.CompletionItemKindKeyword

	var items []protocol.CompletionItem
	for _, name := range styleAttributes {
		if !defined[name] {
			items = append(items, protocol.CompletionItem{
				Label: name,
				Kind:  &kind,
			})
		}
	}

	return items
}

// findDefinedAttributes scans the current block (from the nearest opening brace
// before cursorLine to cursorLine) and returns attribute names already defined
// (lines containing "name = ...").
func findDefinedAttributes(lines []string, cursorLine int) map[string]bool {
	defined := make(map[string]bool)

	// Scan backwards to find the opening brace of the current block
	startLine := 0
	depth := 0
	for i := cursorLine; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		closes := strings.Count(line, "}")
		opens := strings.Count(line, "{")
		depth += closes - opens
		if depth < 0 {
			startLine = i
			break
		}
	}

	// Scan forward from startLine to cursorLine, collecting attribute names
	for i := startLine; i <= cursorLine; i++ {
		line := strings.TrimSpace(lines[i])
		if eqIdx := strings.Index(line, "="); eqIdx > 0 {
			name := strings.TrimSpace(line[:eqIdx])
			if !strings.Contains(name, " ") && !strings.Contains(name, "{") {
				defined[name] = true
			}
		}
	}

	return defined
}

// topLevelCompletions returns completion items for top-level block names.
func topLevelCompletions() []protocol.CompletionItem {
	snippetFormat := protocol.InsertTextFormatSnippet
	kind := protocol.CompletionItemKindSnippet

	var items []protocol.CompletionItem
	for _, name := range topLevelBlocks {
		snippet := name + " {\n  $0\n}"
		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             &kind,
			InsertText:       &snippet,
			InsertTextFormat: &snippetFormat,
		})
	}

	return items
}

// completionKindPtr returns a pointer to a CompletionItemKind.
func completionKindPtr(k protocol.CompletionItemKind) *protocol.CompletionItemKind {
	return &k
}

// textDocumentCompletion is the LSP handler for textDocument/completion requests.
func (s *Server) textDocumentCompletion(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	uri := string(params.TextDocument.URI)

	content, ok := s.docs.Get(uri)
	if !ok {
		return nil, nil
	}

	result := s.getResult(uri)
	if result == nil {
		return nil, nil
	}

	items := complete(result, content, params.Position)
	return items, nil
}
