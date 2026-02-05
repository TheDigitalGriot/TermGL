// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"github.com/charmbracelet/termgl/canvas"
)

// Texture represents a 2D texture of characters and colors.
// Used for texture mapping in shaders.
type Texture struct {
	Width  int
	Height int
	Chars  []rune
	Colors []canvas.Cell
}

// NewTexture creates a new texture with the given dimensions.
// Initializes all cells to spaces with white foreground.
func NewTexture(width, height int) *Texture {
	size := width * height
	chars := make([]rune, size)
	colors := make([]canvas.Cell, size)

	defaultCell := canvas.NewCell(' ', canvas.White)
	for i := 0; i < size; i++ {
		chars[i] = ' '
		colors[i] = defaultCell
	}

	return &Texture{
		Width:  width,
		Height: height,
		Chars:  chars,
		Colors: colors,
	}
}

// NewTextureFromStrings creates a texture from an array of strings.
// Each string is a row of the texture. All rows should have the same length.
func NewTextureFromStrings(rows []string, color canvas.Color) *Texture {
	if len(rows) == 0 {
		return NewTexture(0, 0)
	}

	height := len(rows)
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len([]rune(row))
		}
	}

	tex := NewTexture(width, height)
	cell := canvas.NewCell(' ', color)

	for y, row := range rows {
		for x, r := range row {
			tex.SetChar(x, y, r)
			cell.Rune = r
			tex.SetCell(x, y, cell)
		}
	}

	return tex
}

// SetChar sets the character at the given position.
func (t *Texture) SetChar(x, y int, r rune) {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return
	}
	t.Chars[y*t.Width+x] = r
}

// SetCell sets the full cell (character + color) at the given position.
func (t *Texture) SetCell(x, y int, cell canvas.Cell) {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return
	}
	idx := y*t.Width + x
	t.Chars[idx] = cell.Rune
	t.Colors[idx] = cell
}

// GetChar returns the character at the given position.
func (t *Texture) GetChar(x, y int) rune {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return ' '
	}
	return t.Chars[y*t.Width+x]
}

// GetCell returns the full cell at the given position.
func (t *Texture) GetCell(x, y int) canvas.Cell {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return canvas.DefaultCell()
	}
	return t.Colors[y*t.Width+x]
}

// SampleNearest samples the texture using nearest-neighbor interpolation.
// UV coordinates are in range 0-255.
func (t *Texture) SampleNearest(u, v uint8) (rune, canvas.Cell) {
	if t.Width == 0 || t.Height == 0 {
		return ' ', canvas.DefaultCell()
	}

	// Map UV (0-255) to texture coordinates
	x := int(u) * (t.Width - 1) / 255
	y := int(v) * (t.Height - 1) / 255

	return t.GetChar(x, y), t.GetCell(x, y)
}

// SampleNearestFloat samples with float UV coordinates (0.0-1.0).
func (t *Texture) SampleNearestFloat(u, v float64) (rune, canvas.Cell) {
	// Clamp to 0-1
	if u < 0 {
		u = 0
	}
	if u > 1 {
		u = 1
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}

	return t.SampleNearest(uint8(u*255), uint8(v*255))
}

// TextureShader creates a pixel shader that samples from a texture.
func TextureShader(tex *Texture) PixelShader {
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		return tex.SampleNearest(u, v)
	}
}

// TextureShaderTiled creates a shader that tiles the texture.
// The texture repeats based on the tileU and tileV factors.
func TextureShaderTiled(tex *Texture, tileU, tileV int) PixelShader {
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		// Apply tiling
		tu := uint8((int(u) * tileU) % 256)
		tv := uint8((int(v) * tileV) % 256)
		return tex.SampleNearest(tu, tv)
	}
}

// TextureShaderFlipped creates a shader that can flip the texture horizontally/vertically.
func TextureShaderFlipped(tex *Texture, flipU, flipV bool) PixelShader {
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		if flipU {
			u = 255 - u
		}
		if flipV {
			v = 255 - v
		}
		return tex.SampleNearest(u, v)
	}
}

// ColorMapTexture creates a texture that maps UV to colors.
// Useful for creating color gradients or palettes.
func ColorMapTexture(width, height int, colorFunc func(u, v float64) canvas.Color) *Texture {
	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			u := float64(x) / float64(width-1)
			v := float64(y) / float64(height-1)
			color := colorFunc(u, v)
			tex.SetCell(x, y, canvas.NewCell('█', color))
		}
	}

	return tex
}

// CharMapTexture creates a texture that maps UV to characters from a gradient.
func CharMapTexture(width, height int, grad *Gradient, color canvas.Color) *Texture {
	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			u := float64(x) / float64(width-1)
			char := grad.CharFloat(u)
			tex.SetCell(x, y, canvas.NewCell(char, color))
		}
	}

	return tex
}

// CheckerTexture creates a checkerboard pattern texture.
func CheckerTexture(width, height int, cell1, cell2 canvas.Cell) *Texture {
	tex := NewTexture(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)%2 == 0 {
				tex.SetCell(x, y, cell1)
			} else {
				tex.SetCell(x, y, cell2)
			}
		}
	}

	return tex
}

// NoiseTexture creates a texture with random characters from a gradient.
func NoiseTexture(width, height int, grad *Gradient, color canvas.Color, seed int64) *Texture {
	tex := NewTexture(width, height)

	// Simple LCG random number generator
	rng := seed
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// LCG step
			rng = (rng*1103515245 + 12345) & 0x7fffffff
			intensity := uint8(rng % 256)
			char := grad.Char(intensity)
			tex.SetCell(x, y, canvas.NewCell(char, color))
		}
	}

	return tex
}
