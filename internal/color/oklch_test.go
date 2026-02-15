package color

import (
	"math"
	"testing"
)

func absDiffUint8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestRGBToOKLCH_KnownColors(t *testing.T) {
	tests := []struct {
		name       string
		color      Color
		wantL      float64
		wantC      float64
		wantH      float64
		tolL       float64
		tolC       float64
		tolH       float64
		achromatic bool // skip hue check when C < 0.01
	}{
		{
			name:       "black",
			color:      Color{0, 0, 0},
			wantL:      0.0,
			wantC:      0.0,
			wantH:      0.0,
			tolL:       0.01,
			tolC:       0.01,
			achromatic: true,
		},
		{
			name:       "white",
			color:      Color{255, 255, 255},
			wantL:      1.0,
			wantC:      0.0,
			wantH:      0.0,
			tolL:       0.01,
			tolC:       0.01,
			achromatic: true,
		},
		{
			name:  "red",
			color: Color{255, 0, 0},
			wantL: 0.6279,
			wantC: 0.2577,
			wantH: 29.23,
			tolL:  0.01,
			tolC:  0.01,
			tolH:  0.6,
		},
		{
			name:  "green (0,128,0)",
			color: Color{0, 128, 0},
			wantL: 0.5196,
			wantC: 0.1766,
			wantH: 142.50,
			tolL:  0.01,
			tolC:  0.01,
			tolH:  0.6,
		},
		{
			name:  "blue",
			color: Color{0, 0, 255},
			wantL: 0.4520,
			wantC: 0.3132,
			wantH: 264.05,
			tolL:  0.01,
			tolC:  0.01,
			tolH:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, c, h := RGBToOKLCH(tt.color)

			if math.Abs(l-tt.wantL) > tt.tolL {
				t.Errorf("L = %f, want %f (tol %f)", l, tt.wantL, tt.tolL)
			}
			if math.Abs(c-tt.wantC) > tt.tolC {
				t.Errorf("C = %f, want %f (tol %f)", c, tt.wantC, tt.tolC)
			}
			if !tt.achromatic {
				if math.Abs(h-tt.wantH) > tt.tolH {
					t.Errorf("H = %f, want %f (tol %f)", h, tt.wantH, tt.tolH)
				}
			}
		})
	}
}

func TestOKLCHToRGB_KnownColors(t *testing.T) {
	tests := []struct {
		name  string
		l     float64
		c     float64
		h     float64
		wantR uint8
		wantG uint8
		wantB uint8
	}{
		{"black", 0, 0, 0, 0, 0, 0},
		{"white", 1, 0, 0, 255, 255, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OKLCHToRGB(tt.l, tt.c, tt.h)
			if absDiffUint8(got.R, tt.wantR) > 1 {
				t.Errorf("R = %d, want %d", got.R, tt.wantR)
			}
			if absDiffUint8(got.G, tt.wantG) > 1 {
				t.Errorf("G = %d, want %d", got.G, tt.wantG)
			}
			if absDiffUint8(got.B, tt.wantB) > 1 {
				t.Errorf("B = %d, want %d", got.B, tt.wantB)
			}
		})
	}
}

func TestRGBToOKLCH_Roundtrip(t *testing.T) {
	colors := []Color{
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{235, 111, 146},
		{49, 116, 143},
		{156, 207, 216},
	}

	for _, c := range colors {
		t.Run(c.Hex(), func(t *testing.T) {
			l, ch, h := RGBToOKLCH(c)
			got := OKLCHToRGB(l, ch, h)

			if absDiffUint8(got.R, c.R) > 1 {
				t.Errorf("R = %d, want %d", got.R, c.R)
			}
			if absDiffUint8(got.G, c.G) > 1 {
				t.Errorf("G = %d, want %d", got.G, c.G)
			}
			if absDiffUint8(got.B, c.B) > 1 {
				t.Errorf("B = %d, want %d", got.B, c.B)
			}
		})
	}
}
