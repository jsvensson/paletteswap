package color

import "math"

// RGBToOKLCH converts an sRGB Color to OKLCH components.
// L is lightness [0, 1], chroma is colorfulness [0, ~0.37], hue is in degrees [0, 360).
func RGBToOKLCH(c Color) (l, chroma, hue float64) {
	// sRGB → linear RGB
	lr := srgbToLinear(float64(c.R) / 255.0)
	lg := srgbToLinear(float64(c.G) / 255.0)
	lb := srgbToLinear(float64(c.B) / 255.0)

	// linear RGB → OKLAB
	L, a, b := linearRGBToOKLAB(lr, lg, lb)

	// OKLAB → OKLCH
	chroma = math.Sqrt(a*a + b*b)
	hue = math.Atan2(b, a) * (180.0 / math.Pi)
	if hue < 0 {
		hue += 360.0
	}

	return L, chroma, hue
}

// OKLCHToRGB converts OKLCH components to an sRGB Color.
// L is lightness [0, 1], chroma is colorfulness, hue is in degrees [0, 360).
func OKLCHToRGB(l, chroma, hue float64) Color {
	// OKLCH → OKLAB
	hRad := hue * (math.Pi / 180.0)
	a := chroma * math.Cos(hRad)
	b := chroma * math.Sin(hRad)

	// OKLAB → linear RGB
	lr, lg, lb := oklabToLinearRGB(l, a, b)

	// linear RGB → sRGB, clamped
	r := linearToSRGB(clamp01(lr))
	g := linearToSRGB(clamp01(lg))
	bl := linearToSRGB(clamp01(lb))

	return Color{
		R: uint8(math.Round(r * 255.0)),
		G: uint8(math.Round(g * 255.0)),
		B: uint8(math.Round(bl * 255.0)),
	}
}

// srgbToLinear converts a single sRGB component [0,1] to linear RGB.
func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// linearToSRGB converts a single linear RGB component [0,1] to sRGB.
func linearToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

// linearRGBToOKLAB converts linear RGB to OKLAB (L, a, b).
func linearRGBToOKLAB(r, g, b float64) (float64, float64, float64) {
	// M1: linear RGB → LMS
	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	// Cube root (preserving sign)
	lp := math.Cbrt(l)
	mp := math.Cbrt(m)
	sp := math.Cbrt(s)

	// M2: LMS' → Lab
	L := 0.2104542553*lp + 0.7936177850*mp - 0.0040720468*sp
	A := 1.9779984951*lp - 2.4285922050*mp + 0.4505937099*sp
	B := 0.0259040371*lp + 0.7827717662*mp - 0.8086757660*sp

	return L, A, B
}

// oklabToLinearRGB converts OKLAB (L, a, b) to linear RGB.
func oklabToLinearRGB(L, a, b float64) (float64, float64, float64) {
	// Inverse M2: Lab → LMS'
	lp := L + 0.3963377774*a + 0.2158037573*b
	mp := L - 0.1055613458*a - 0.0638541728*b
	sp := L - 0.0894841775*a - 1.2914855480*b

	// Cube: LMS' → LMS
	l := lp * lp * lp
	m := mp * mp * mp
	s := sp * sp * sp

	// Inverse M1: LMS → linear RGB
	r := +4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	bl := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s

	return r, g, bl
}

// StepLightness returns a new Color with the given absolute OKLCH lightness,
// preserving the original color's hue and chroma. Lightness should be in [0, 1].
func StepLightness(c Color, lightness float64) Color {
	_, chroma, hue := RGBToOKLCH(c)
	return OKLCHToRGB(lightness, chroma, hue)
}

// clamp01 clamps a value to the [0, 1] range.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
