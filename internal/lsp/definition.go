package lsp

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// blockRefAtCursor extracts the block reference path up to the cursor position.
// Works with any block: palette, theme, ansi, syntax.
// For example, if cursor is on "palette" in "palette.base", it returns "palette".
// If cursor is on "base" in "palette.base", it returns "palette.base".
// Returns "" if the cursor is not on a block reference.
func blockRefAtCursor(line string, character uint32) string {
	col := int(character)
	if col >= len(line) {
		return ""
	}

	// Find the end of the current word (letters, digits, underscores, dots)
	end := col
	for end < len(line) && isIdentChar(line[end]) {
		end++
	}

	// Find the start of the current word (letters, digits, underscores, dots)
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

// isIdentChar returns true if the byte is a valid identifier character
// (letter, digit, underscore, or dot for dotted paths).
func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' || b == '.'
}

// definition returns the definition location for a palette reference at the given cursor position.
// It extracts the palette path from the current line, looks it up in the symbol table,
// and returns the location of its definition. Returns nil if the cursor is not on a palette reference
// or if the symbol is not found.
func definition(result *AnalysisResult, content string, uri string, pos protocol.Position) *protocol.Location {
	if result == nil {
		return nil
	}

	lines := strings.Split(content, "\n")
	lineIdx := int(pos.Line)
	if lineIdx >= len(lines) {
		return nil
	}

	line := lines[lineIdx]
	ref := blockRefAtCursor(line, pos.Character)
	if ref == "" {
		return nil
	}

	symRange, ok := result.Symbols[ref]
	if !ok {
		return nil
	}

	return &protocol.Location{
		URI:   protocol.DocumentUri(uri),
		Range: symRange,
	}
}

// textDocumentDefinition handles textDocument/definition requests.
func (s *Server) textDocumentDefinition(_ *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	uri := string(params.TextDocument.URI)

	result := s.getResult(uri)
	if result == nil {
		return nil, nil
	}

	content, ok := s.docs.Get(uri)
	if !ok {
		return nil, nil
	}

	return definition(result, content, uri, params.Position), nil
}
