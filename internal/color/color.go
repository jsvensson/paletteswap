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

// ColorTree represents a nested map of colors, used for syntax scopes.
// Values are either Style or ColorTree.
type ColorTree map[string]any

// IsStyle returns true if the value is a Style (leaf node), false if it's a ColorTree.
func IsStyle(v any) bool {
	_, ok := v.(Style)
	return ok
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

// RGB returns the color as an rgb() string, e.g. "rgb(235, 111, 146)".
func (c Color) RGB() string {
	return fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B)
}

// RGBA returns the color in rgba() function format with full opacity
func (c Color) RGBA() string {
	return fmt.Sprintf("rgba(%d, %d, %d, 1.0)", c.R, c.G, c.B)
}
