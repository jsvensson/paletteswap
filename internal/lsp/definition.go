package lsp

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// paletteRefAtCursor extracts the full palette reference (e.g. "palette.highlight.low")
// at the given cursor position in a line, or returns "" if the cursor is not on a palette reference.
func paletteRefAtCursor(line string, character uint32) string {
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
	if !strings.HasPrefix(word, "palette.") {
		return ""
	}

	return word
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
	ref := paletteRefAtCursor(line, pos.Character)
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
