package tier1

import (
	"image"
	"image/color"
	"math"
)

// Quantizer reduces a truecolor image to a paletted image.
// Architecture doc Section 4.3
type Quantizer interface {
	Quantize(img *image.NRGBA, numColors int) *image.Paletted
}

// DitherMode affects animation quality.
// Architecture doc Section 4.3
type DitherMode int

const (
	DitherNone           DitherMode = iota // No dithering (fastest, visible banding)
	DitherFloydSteinberg                   // Error diffusion (best quality, temporal noise)
	DitherOrdered4x4                       // Bayer 4x4 (good for animation, no crawl)
	DitherOrdered8x8                       // Bayer 8x8 (smoother gradients, subtle pattern)
)

// ColorSpace for distance calculations during quantization.
// Architecture doc Section 4.3
type ColorSpace int

const (
	ColorSpaceRGB    ColorSpace = iota // Fastest, worst perceptual accuracy
	ColorSpaceCIELAB                   // Good perceptual accuracy, moderate cost
	ColorSpaceDIN99d                   // Best perceptual uniformity, highest cost
)

// Bayer dither matrices (normalized to 0-1)
var bayer4x4 = [4][4]float64{
	{0.0 / 16, 8.0 / 16, 2.0 / 16, 10.0 / 16},
	{12.0 / 16, 4.0 / 16, 14.0 / 16, 6.0 / 16},
	{3.0 / 16, 11.0 / 16, 1.0 / 16, 9.0 / 16},
	{15.0 / 16, 7.0 / 16, 13.0 / 16, 5.0 / 16},
}

var bayer8x8 = [8][8]float64{
	{0.0 / 64, 32.0 / 64, 8.0 / 64, 40.0 / 64, 2.0 / 64, 34.0 / 64, 10.0 / 64, 42.0 / 64},
	{48.0 / 64, 16.0 / 64, 56.0 / 64, 24.0 / 64, 50.0 / 64, 18.0 / 64, 58.0 / 64, 26.0 / 64},
	{12.0 / 64, 44.0 / 64, 4.0 / 64, 36.0 / 64, 14.0 / 64, 46.0 / 64, 6.0 / 64, 38.0 / 64},
	{60.0 / 64, 28.0 / 64, 52.0 / 64, 20.0 / 64, 62.0 / 64, 30.0 / 64, 54.0 / 64, 22.0 / 64},
	{3.0 / 64, 35.0 / 64, 11.0 / 64, 43.0 / 64, 1.0 / 64, 33.0 / 64, 9.0 / 64, 41.0 / 64},
	{51.0 / 64, 19.0 / 64, 59.0 / 64, 27.0 / 64, 49.0 / 64, 17.0 / 64, 57.0 / 64, 25.0 / 64},
	{15.0 / 64, 47.0 / 64, 7.0 / 64, 39.0 / 64, 13.0 / 64, 45.0 / 64, 5.0 / 64, 37.0 / 64},
	{63.0 / 64, 31.0 / 64, 55.0 / 64, 23.0 / 64, 61.0 / 64, 29.0 / 64, 53.0 / 64, 21.0 / 64},
}

// colorDistance computes the squared distance between two colors in the specified color space.
func colorDistance(c1, c2 color.NRGBA, space ColorSpace) float64 {
	switch space {
	case ColorSpaceCIELAB:
		return colorDistanceCIELAB(c1, c2)
	case ColorSpaceDIN99d:
		return colorDistanceDIN99d(c1, c2)
	default:
		return colorDistanceRGB(c1, c2)
	}
}

// colorDistanceRGB computes squared Euclidean distance in RGB space.
func colorDistanceRGB(c1, c2 color.NRGBA) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return dr*dr + dg*dg + db*db
}

// colorDistanceCIELAB computes squared Euclidean distance in CIELAB color space.
func colorDistanceCIELAB(c1, c2 color.NRGBA) float64 {
	l1, a1, b1 := rgbToCIELAB(c1)
	l2, a2, b2 := rgbToCIELAB(c2)
	dl := l1 - l2
	da := a1 - a2
	db := b1 - b2
	return dl*dl + da*da + db*db
}

// colorDistanceDIN99d computes squared Euclidean distance in DIN99d color space.
func colorDistanceDIN99d(c1, c2 color.NRGBA) float64 {
	l1, a1, b1 := rgbToDIN99d(c1)
	l2, a2, b2 := rgbToDIN99d(c2)
	dl := l1 - l2
	da := a1 - a2
	db := b1 - b2
	return dl*dl + da*da + db*db
}

// rgbToCIELAB converts sRGB to CIELAB color space.
// Uses D65 illuminant (standard daylight).
func rgbToCIELAB(c color.NRGBA) (l, a, b float64) {
	r := srgbToLinear(float64(c.R) / 255.0)
	g := srgbToLinear(float64(c.G) / 255.0)
	bl := srgbToLinear(float64(c.B) / 255.0)

	// Linear RGB to XYZ (D65 illuminant)
	x := r*0.4124564 + g*0.3575761 + bl*0.1804375
	y := r*0.2126729 + g*0.7151522 + bl*0.0721750
	z := r*0.0193339 + g*0.1191920 + bl*0.9503041

	// D65 white point
	const xn, yn, zn = 0.95047, 1.00000, 1.08883

	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	l = 116.0*fy - 16.0
	a = 500.0 * (fx - fy)
	b = 200.0 * (fy - fz)
	return
}

// rgbToDIN99d converts sRGB to DIN99d color space.
func rgbToDIN99d(c color.NRGBA) (l99, a99, b99 float64) {
	lLab, aLab, bLab := rgbToCIELAB(c)

	const cos16 = 0.9612616959 // math.Cos(16 * math.Pi / 180)
	const sin16 = 0.2756373558 // math.Sin(16 * math.Pi / 180)

	l99 = 105.51*math.Log(1.0+0.0158*lLab) // 325.22 * f(L*/100) approximation

	e := aLab*cos16 + bLab*sin16
	f := 0.7 * (bLab*cos16 - aLab*sin16)

	g := math.Sqrt(e*e + f*f)
	if g < 1e-10 {
		a99 = 0
		b99 = 0
		return
	}

	cc := math.Log(1.0 + 0.045*g) / 0.045

	a99 = cc * e / g
	b99 = cc * f / g
	return
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	const deltaCubed = delta * delta * delta

	if t > deltaCubed {
		return math.Cbrt(t)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

// clampF restricts a float64 value to the range [lo, hi].
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// GetBayer4x4 returns the Bayer 4x4 threshold value at the given pixel position.
func GetBayer4x4(x, y int) float64 {
	return bayer4x4[y%4][x%4]
}

// GetBayer8x8 returns the Bayer 8x8 threshold value at the given pixel position.
func GetBayer8x8(x, y int) float64 {
	return bayer8x8[y%8][x%8]
}
