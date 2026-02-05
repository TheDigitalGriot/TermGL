// Package gl provides 3D rendering pipeline for terminal graphics.
package gl

import (
	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/draw"
)

// Texture is an alias for draw.Texture for 3D rendering.
// Provides UV-based texture sampling for the shader pipeline.
type Texture = draw.Texture

// NewTexture creates a new texture with the given dimensions.
// All cells are initialized to spaces with white foreground.
func NewTexture(width, height int) *Texture {
	return draw.NewTexture(width, height)
}

// NewTextureFromStrings creates a texture from an array of strings.
// Each string is a row of the texture. All rows should have the same length.
// The color is applied to all cells.
func NewTextureFromStrings(rows []string, color canvas.Color) *Texture {
	return draw.NewTextureFromStrings(rows, color)
}

// NewTextureFromCharsAndColors creates a texture with individual character colors.
// rows contains the character data, colors maps characters to their colors.
func NewTextureFromCharsAndColors(rows []string, colors map[rune]canvas.Color) *Texture {
	if len(rows) == 0 {
		return NewTexture(0, 0)
	}

	height := len(rows)
	width := 0
	for _, row := range rows {
		if len([]rune(row)) > width {
			width = len([]rune(row))
		}
	}

	tex := NewTexture(width, height)

	for y, row := range rows {
		for x, char := range row {
			color, ok := colors[char]
			if !ok {
				color = canvas.White
			}
			tex.SetCell(x, y, canvas.NewCell(char, color))
		}
	}

	return tex
}

// TextureShader creates a PixelShader that samples from a texture.
// This is a convenience function for creating textured rendering.
func TextureShader(tex *Texture) PixelShader {
	data := &PixelShaderTexture{Texture: tex}
	return func(u, v uint8, _ any) (rune, canvas.Color, canvas.Color) {
		return PixelShaderTextureFunc(u, v, data)
	}
}
