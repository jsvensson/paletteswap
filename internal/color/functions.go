package color

import "math"

// Brighten returns a brighter version of the given color.
func Brighten(color Color, percentage float64) Color {
	// Normalize RGB to 0-1 range
	r, g, b := float64(color.R)/255.0, float64(color.G)/255.0, float64(color.B)/255.0

	// Convert RGB to HSL internally
	var h, s, l float64

	min := math.Min(math.Min(r, g), b)
	max := math.Max(math.Max(r, g), b)
	l = (max + min) / 2.0

	if max == min {
		h = 0
		s = 0 // Achromatic
	} else {
		d := max - min
		if l > 0.5 {
			s = d / (2.0 - max - min)
		} else {
			s = d / (max + min)
		}

		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6.0
			}
			h /= 6.0
		case g:
			h = ((b-r)/d + 2.0) / 6.0
		case b:
			h = ((r-g)/d + 4.0) / 6.0
		}

	}

	// Increase lightness
	l = math.Min(1.0, l+(percentage))

	// Convert back to RGB
	var r1, g1, b1 float64

	if s == 0 { // Achromatic
		r1, g1, b1 = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1.0 + s)
		} else {
			q = l + s - l*s
		}
		p := 2.0*l - q

		r1 = hueToRGB(p, q, h+1.0/3.0)
		g1 = hueToRGB(p, q, h)
		b1 = hueToRGB(p, q, h-1.0/3.0)
	}

	return Color{
		R: uint8(r1 * 255),
		G: uint8(g1 * 255),
		B: uint8(b1 * 255),
	}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1.0
	}
	if t > 1 {
		t -= 1.0
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6.0*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6.0
	}
	return p

}
