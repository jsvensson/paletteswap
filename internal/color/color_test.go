package color

import (
	"testing"
)

func TestParseHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Color
		wantErr bool
	}{
		{"with hash", "#eb6f92", Color{235, 111, 146}, false},
		{"without hash", "eb6f92", Color{235, 111, 146}, false},
		{"black", "#000000", Color{0, 0, 0}, false},
		{"white", "#ffffff", Color{255, 255, 255}, false},
		{"uppercase", "#AABBCC", Color{170, 187, 204}, false},
		{"too short", "#fff", Color{}, true},
		{"too long", "#aabbccdd", Color{}, true},
		{"invalid chars", "#zzzzzz", Color{}, true},
		{"empty", "", Color{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHex(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseHex(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestColorHex(t *testing.T) {
	c := Color{235, 111, 146}
	want := "#eb6f92"
	if got := c.Hex(); got != want {
		t.Errorf("Color.Hex() = %q, want %q", got, want)
	}
}

func TestColorHexBare(t *testing.T) {
	c := Color{235, 111, 146}
	want := "eb6f92"
	if got := c.HexBare(); got != want {
		t.Errorf("Color.HexBare() = %q, want %q", got, want)
	}
}

func TestColorRGB(t *testing.T) {
	c := Color{235, 111, 146}
	want := "rgb(235, 111, 146)"
	if got := c.RGB(); got != want {
		t.Errorf("Color.RGB() = %q, want %q", got, want)
	}
}

func TestColorHexZeroPadding(t *testing.T) {
	c := Color{0, 5, 10}
	want := "#00050a"
	if got := c.Hex(); got != want {
		t.Errorf("Color.Hex() = %q, want %q", got, want)
	}
}

func TestBrighten(t *testing.T) {
	tests := []struct {
		name       string
		color      Color
		percentage float64
		want       Color
	}{
		{
			name:       "brighten red by 10%",
			color:      Color{255, 0, 0},
			percentage: 0.1,
			want:       Color{255, 50, 50},
		},
		{
			name:       "brighten gray by 20%",
			color:      Color{128, 128, 128},
			percentage: 0.2,
			want:       Color{179, 179, 179},
		},
		{
			name:       "white stays white",
			color:      Color{255, 255, 255},
			percentage: 0.5,
			want:       Color{255, 255, 255},
		},
		{
			name:       "brighten black by 50%",
			color:      Color{0, 0, 0},
			percentage: 0.5,
			want:       Color{127, 127, 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Brighten(tt.color, tt.percentage)
			if got != tt.want {
				t.Errorf("Brighten(%v, %v) = %v, want %v", tt.color, tt.percentage, got, tt.want)
			}
		})
	}
}

func TestColor_RGBA(t *testing.T) {
	tests := []struct {
		name     string
		color    Color
		expected string
	}{
		{
			name:     "red with full opacity",
			color:    Color{255, 0, 0},
			expected: "rgba(255, 0, 0, 1.0)",
		},
		{
			name:     "green with full opacity",
			color:    Color{0, 255, 0},
			expected: "rgba(0, 255, 0, 1.0)",
		},
		{
			name:     "dark color",
			color:    Color{25, 23, 36},
			expected: "rgba(25, 23, 36, 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.RGBA()
			if got != tt.expected {
				t.Errorf("RGBA() = %v, want %v", got, tt.expected)
			}
		})
	}
}
