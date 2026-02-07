package paletteswap

import (
	"fmt"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/parser"
)

// Theme is the fully-resolved theme data, ready for template rendering.
type Theme struct {
	Meta    Meta
	Palette *color.Node
	Syntax  color.Tree
	Theme   map[string]color.Color
	ANSI    map[string]color.Color
}

// Meta holds theme metadata.
type Meta struct {
	Name       string
	Author     string
	Appearance string
	URL        string
}

// Load parses an HCL theme file and returns a fully-resolved Theme.
func Load(path string) (*Theme, error) {
	raw, err := parser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("loading theme: %w", err)
	}

	return &Theme{
		Meta: Meta{
			Name:       raw.Meta.Name,
			Author:     raw.Meta.Author,
			Appearance: raw.Meta.Appearance,
			URL:        raw.Meta.URL,
		},
		Palette: raw.Palette,
		Theme:   raw.Theme,
		Syntax:  raw.Syntax,
		ANSI:    raw.ANSI,
	}, nil
}
