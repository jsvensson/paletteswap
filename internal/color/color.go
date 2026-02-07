package color

import (
	"fmt"
	"strings"
)

// Color represents an RGB color. The R, G, B uint8 fields are the source of truth;
// all output formats are derived from them.
type Color struct {
	R, G, B uint8
}

// Style represents a syntax scope entry with a color and optional font styles.
type Style struct {
	Color     Color
	Bold      bool
	Italic    bool
	Underline bool
}

// Tree represents a nested map of colors, used for syntax scopes.
// Values are either Style or Tree.
type Tree map[string]any

// Node represents a palette entry that can be both a color and a namespace.
// Color is nil for namespace-only nodes (groups without a color attribute).
// Children is nil for leaf nodes (flat color attributes).
type Node struct {
	Color    *Color
	Children map[string]*Node
}

// Lookup resolves a dot-path (as segments) to a Color.
// Returns an error if the path is not found or the target node has no color.
func (n *Node) Lookup(path []string) (Color, error) {
	current := n
	for _, part := range path {
		if current.Children == nil {
			return Color{}, fmt.Errorf("path not found: %s is a leaf, cannot traverse further", part)
		}
		child, ok := current.Children[part]
		if !ok {
			return Color{}, fmt.Errorf("path not found: %q does not exist", part)
		}
		current = child
	}
	if current.Color == nil {
		return Color{}, fmt.Errorf("path is a group, not a color; add a color attribute or reference a specific child")
	}
	return *current.Color, nil
}

// ParseHex parses a hex color string like "#eb6f92" into a Color.
func ParseHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return Color{}, fmt.Errorf("invalid hex color %q: must be 6 hex digits", s)
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return Color{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return Color{R: r, G: g, B: b}, nil
}

// Hex returns the color as a hex string with leading #, e.g. "#eb6f92".
func (c Color) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// HexBare returns the color as a hex string without leading #, e.g. "eb6f92".
func (c Color) HexBare() string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

// HexAlpha returns the color in hex format with alpha channel (#rrggbbaa)
func (c Color) HexAlpha() string {
	return c.Hex() + "ff"
}

// HexBareAlpha returns the color in hex format without # prefix and with alpha channel (rrggbbaa)
func (c Color) HexBareAlpha() string {
	return c.HexBare() + "ff"
}

// RGB returns the color as an rgb() string, e.g. "rgb(235, 111, 146)".
func (c Color) RGB() string {
	return fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B)
}

// RGBA returns the color in rgba() function format with full opacity
func (c Color) RGBA() string {
	return fmt.Sprintf("rgba(%d, %d, %d, 1.0)", c.R, c.G, c.B)
}
