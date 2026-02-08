package tier1

import (
	"image"
	"image/color"
)

// FixedPaletteQuantizer uses a pre-computed 3-3-2 RGB palette.
// Zero per-frame palette computation. Fastest possible quantization.
// Inspired by ref/impulse-gerp-2025.
type FixedPaletteQuantizer struct {
	palette color.Palette
	lookup  [256 * 256 * 256 / 1024]uint8 // Approximate lookup (quantized to 6-bit per channel)
	Dither  DitherMode
}

// NewFixedPaletteQuantizer creates a quantizer with a fixed 3-3-2 RGB palette.
// 3 bits for red (8 levels), 3 bits for green (8 levels), 2 bits for blue (4 levels) = 256 colors.
func NewFixedPaletteQuantizer(dither DitherMode) *FixedPaletteQuantizer {
	q := &FixedPaletteQuantizer{
		palette: make(color.Palette, 256),
		Dither:  dither,
	}

	// Build 3-3-2 palette
	for i := 0; i < 256; i++ {
		// Extract 3-3-2 components
		r3 := (i >> 5) & 0x07 // 3 bits red
		g3 := (i >> 2) & 0x07 // 3 bits green
		b2 := i & 0x03        // 2 bits blue

		// Scale to 0-255 range
		r := uint8(r3 * 255 / 7)
		g := uint8(g3 * 255 / 7)
		b := uint8(b2 * 255 / 3)

		q.palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}

	return q
}

// colorToIndex332 maps an NRGBA color to a 3-3-2 palette index.
func colorToIndex332(c color.NRGBA) uint8 {
	r3 := c.R >> 5        // Top 3 bits of red
	g3 := c.G >> 5        // Top 3 bits of green
	b2 := c.B >> 6        // Top 2 bits of blue
	return r3<<5 | g3<<2 | b2
}

// Quantize reduces an NRGBA image to a paletted image using the fixed 3-3-2 palette.
func (q *FixedPaletteQuantizer) Quantize(img *image.NRGBA, numColors int) *image.Paletted {
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, q.palette)
	width, height := bounds.Dx(), bounds.Dy()

	switch q.Dither {
	case DitherFloydSteinberg:
		q.ditherFloydSteinberg(img, paletted, width, height, bounds)
	case DitherOrdered4x4:
		q.ditherOrdered(img, paletted, width, height, bounds, 4)
	case DitherOrdered8x8:
		q.ditherOrdered(img, paletted, width, height, bounds, 8)
	default:
		// No dithering — direct mapping (fastest path)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
				paletted.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, colorToIndex332(c))
			}
		}
	}

	return paletted
}

func (q *FixedPaletteQuantizer) ditherFloydSteinberg(src *image.NRGBA, dst *image.Paletted, width, height int, bounds image.Rectangle) {
	errors := make([][]struct{ r, g, b float64 }, height)
	for y := range errors {
		errors[y] = make([]struct{ r, g, b float64 }, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			r := clamp(float64(c.R)+errors[y][x].r, 0, 255)
			g := clamp(float64(c.G)+errors[y][x].g, 0, 255)
			b := clamp(float64(c.B)+errors[y][x].b, 0, 255)

			adjusted := color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			idx := colorToIndex332(adjusted)
			dst.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, idx)

			palColor := q.palette[idx].(color.NRGBA)
			errR := r - float64(palColor.R)
			errG := g - float64(palColor.G)
			errB := b - float64(palColor.B)

			if x+1 < width {
				errors[y][x+1].r += errR * 7.0 / 16.0
				errors[y][x+1].g += errG * 7.0 / 16.0
				errors[y][x+1].b += errB * 7.0 / 16.0
			}
			if y+1 < height {
				if x > 0 {
					errors[y+1][x-1].r += errR * 3.0 / 16.0
					errors[y+1][x-1].g += errG * 3.0 / 16.0
					errors[y+1][x-1].b += errB * 3.0 / 16.0
				}
				errors[y+1][x].r += errR * 5.0 / 16.0
				errors[y+1][x].g += errG * 5.0 / 16.0
				errors[y+1][x].b += errB * 5.0 / 16.0
				if x+1 < width {
					errors[y+1][x+1].r += errR * 1.0 / 16.0
					errors[y+1][x+1].g += errG * 1.0 / 16.0
					errors[y+1][x+1].b += errB * 1.0 / 16.0
				}
			}
		}
	}
}

func (q *FixedPaletteQuantizer) ditherOrdered(src *image.NRGBA, dst *image.Paletted, width, height int, bounds image.Rectangle, matrixSize int) {
	var matrix [][]float64
	if matrixSize == 4 {
		matrix = make([][]float64, 4)
		for i := range matrix {
			matrix[i] = bayer4x4[i][:]
		}
	} else {
		matrix = make([][]float64, 8)
		for i := range matrix {
			matrix[i] = bayer8x8[i][:]
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			threshold := matrix[y%matrixSize][x%matrixSize]
			r := clamp(float64(c.R)+(threshold-0.5)*32.0, 0, 255)
			g := clamp(float64(c.G)+(threshold-0.5)*32.0, 0, 255)
			b := clamp(float64(c.B)+(threshold-0.5)*32.0, 0, 255)

			adjusted := color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			dst.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, colorToIndex332(adjusted))
		}
	}
}
