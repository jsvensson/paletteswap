package lsp

import (
	"fmt"
	"strings"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// colorToLSP converts an internal color.Color (uint8 RGB) to a protocol.Color (float32 0.0-1.0).
func colorToLSP(c color.Color) protocol.Color {
	return protocol.Color{
		Red:   float32(c.R) / 255.0,
		Green: float32(c.G) / 255.0,
		Blue:  float32(c.B) / 255.0,
		Alpha: 1.0,
	}
}

// documentColors converts the analysis result's color locations into LSP ColorInformation items.
func documentColors(result *AnalysisResult) []protocol.ColorInformation {
	if result == nil {
		return []protocol.ColorInformation{}
	}

	infos := make([]protocol.ColorInformation, 0, len(result.Colors))
	for _, cl := range result.Colors {
		infos = append(infos, protocol.ColorInformation{
			Range: cl.Range,
			Color: colorToLSP(cl.Color),
		})
	}
	return infos
}

// colorPresentation produces color presentation options for a given color and range.
// For hex literals (text starting with `"` or `#`), it returns a presentation with a TextEdit
// to replace the old value. For palette references (text starting with `palette.`), it returns
// an empty slice to avoid replacing references with literal values.
func colorPresentation(content string, params *protocol.ColorPresentationParams) []protocol.ColorPresentation {
	r := uint8(params.Color.Red * 255)
	g := uint8(params.Color.Green * 255)
	b := uint8(params.Color.Blue * 255)
	hexStr := fmt.Sprintf("#%02x%02x%02x", r, g, b)

	// Extract the text at the given range to determine if this is a hex literal or a reference
	text := extractText(content, params.Range)

	if strings.HasPrefix(text, "palette.") {
		// Don't replace palette references with hex literals
		return []protocol.ColorPresentation{}
	}

	if strings.HasPrefix(text, "\"") || strings.HasPrefix(text, "#") {
		// Determine the replacement text: include quotes if the original had them
		newText := hexStr
		if strings.HasPrefix(text, "\"") {
			newText = "\"" + hexStr + "\""
		}

		return []protocol.ColorPresentation{
			{
				Label: hexStr,
				TextEdit: &protocol.TextEdit{
					Range:   params.Range,
					NewText: newText,
				},
			},
		}
	}

	// Unknown format, return empty
	return []protocol.ColorPresentation{}
}

// textDocumentDocumentColor handles textDocument/documentColor requests.
func (s *Server) textDocumentDocumentColor(_ *glsp.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	uri := string(params.TextDocument.URI)
	result := s.getResult(uri)
	return documentColors(result), nil
}

// textDocumentColorPresentation handles textDocument/colorPresentation requests.
func (s *Server) textDocumentColorPresentation(_ *glsp.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	uri := string(params.TextDocument.URI)
	content, ok := s.docs.Get(uri)
	if !ok {
		return []protocol.ColorPresentation{}, nil
	}
	return colorPresentation(content, params), nil
}
