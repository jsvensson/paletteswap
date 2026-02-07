package lsp

import (
	"fmt"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// posInRange returns true if pos is within the range [r.Start, r.End).
// The end position is exclusive.
func posInRange(pos protocol.Position, r protocol.Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}

// extractText extracts the source text at a given LSP range from document content.
func extractText(content string, r protocol.Range) string {
	lines := strings.Split(content, "\n")

	startLine := int(r.Start.Line)
	endLine := int(r.End.Line)

	if startLine >= len(lines) {
		return ""
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	if startLine == endLine {
		line := lines[startLine]
		startChar := int(r.Start.Character)
		endChar := int(r.End.Character)
		if startChar > len(line) {
			startChar = len(line)
		}
		if endChar > len(line) {
			endChar = len(line)
		}
		return line[startChar:endChar]
	}

	// Multi-line range
	var parts []string
	for i := startLine; i <= endLine; i++ {
		line := lines[i]
		if i == startLine {
			startChar := int(r.Start.Character)
			if startChar > len(line) {
				startChar = len(line)
			}
			parts = append(parts, line[startChar:])
		} else if i == endLine {
			endChar := int(r.End.Character)
			if endChar > len(line) {
				endChar = len(line)
			}
			parts = append(parts, line[:endChar])
		} else {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

// hover produces a Hover response for the given cursor position.
// It checks whether the position falls within any ColorLocation from the analysis result.
// For palette references (IsRef=true), the hover shows the source text, hex, and RGB.
// For hex literals, it shows hex and RGB.
// Returns nil if no color is found at the position.
func hover(result *AnalysisResult, content string, pos protocol.Position) *protocol.Hover {
	if result == nil {
		return nil
	}

	for _, cl := range result.Colors {
		if !posInRange(pos, cl.Range) {
			continue
		}

		var md string
		if cl.IsRef {
			sourceText := extractText(content, cl.Range)
			md = fmt.Sprintf("**%s**\n\n`%s` \u00b7 `%s`", sourceText, cl.Color.Hex(), cl.Color.RGB())
		} else {
			md = fmt.Sprintf("`%s` \u00b7 `%s`", cl.Color.Hex(), cl.Color.RGB())
		}

		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: md,
			},
			Range: &cl.Range,
		}
	}

	return nil
}

// textDocumentHover handles textDocument/hover requests.
func (s *Server) textDocumentHover(_ *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := string(params.TextDocument.URI)

	result := s.getResult(uri)
	if result == nil {
		return nil, nil
	}

	content, ok := s.docs.Get(uri)
	if !ok {
		return nil, nil
	}

	return hover(result, content, params.Position), nil
}
