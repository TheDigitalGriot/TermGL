// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"github.com/charmbracelet/termgl/canvas"
)

// PixelShader is a function that determines the character and color for a pixel
// based on its UV coordinates. This allows for custom shading, textures, and effects.
//
// Parameters:
//   - u, v: Texture/interpolation coordinates (0-255), interpolated across the primitive
//   - x, y: Screen coordinates of the pixel
//
// Returns:
//   - rune: The character to draw
//   - canvas.Cell: The full cell styling (color, bold, etc.)
type PixelShader func(u, v uint8, x, y int) (rune, canvas.Cell)

// Vertex2D represents a 2D vertex with position and UV coordinates.
type Vertex2D struct {
	X, Y int     // Screen position
	Z    float64 // Depth (for z-buffering, optional)
	U, V uint8   // UV coordinates (0-255) for shader interpolation
}

// NewVertex2D creates a new 2D vertex.
func NewVertex2D(x, y int, u, v uint8) Vertex2D {
	return Vertex2D{X: x, Y: y, U: u, V: v}
}

// NewVertex2DFloat creates a vertex with float coordinates that get rounded.
func NewVertex2DFloat(x, y float64, u, v uint8) Vertex2D {
	return Vertex2D{
		X: int(x + 0.5),
		Y: int(y + 0.5),
		U: u,
		V: v,
	}
}

// PointShaded draws a single point using a pixel shader.
func PointShaded(c *canvas.Canvas, v Vertex2D, shader PixelShader) {
	r, cell := shader(v.U, v.V, v.X, v.Y)
	cell.Rune = r
	c.SetCell(v.X, v.Y, cell)
}

// LineShaded draws a line with interpolated UV coordinates using a pixel shader.
func LineShaded(c *canvas.Canvas, v0, v1 Vertex2D, shader PixelShader) {
	x1, y1 := v0.X, v0.Y
	x2, y2 := v1.X, v1.Y

	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	steps := max(dx, dy)

	if steps == 0 {
		PointShaded(c, v0, shader)
		return
	}

	xInc := float64(x2-x1) / float64(steps)
	yInc := float64(y2-y1) / float64(steps)
	uInc := float64(int(v1.U)-int(v0.U)) / float64(steps)
	vInc := float64(int(v1.V)-int(v0.V)) / float64(steps)

	x := float64(x1)
	y := float64(y1)
	u := float64(v0.U)
	v := float64(v0.V)

	for i := 0; i <= steps; i++ {
		ix := int(x + 0.5)
		iy := int(y + 0.5)
		iu := clampUint8(int(u + 0.5))
		iv := clampUint8(int(v + 0.5))

		if c.InBounds(ix, iy) {
			r, cell := shader(iu, iv, ix, iy)
			cell.Rune = r
			c.SetCell(ix, iy, cell)
		}

		x += xInc
		y += yInc
		u += uInc
		v += vInc
	}
}

// TriangleShaded draws a filled triangle with interpolated UV using a pixel shader.
func TriangleShaded(c *canvas.Canvas, v0, v1, v2 Vertex2D, shader PixelShader) {
	// Sort vertices by Y coordinate (ascending)
	if v0.Y > v1.Y {
		v0, v1 = v1, v0
	}
	if v1.Y > v2.Y {
		v1, v2 = v2, v1
	}
	if v0.Y > v1.Y {
		v0, v1 = v1, v0
	}

	// Degenerate case
	if v0.Y == v2.Y {
		// Horizontal line
		minX := min(v0.X, min(v1.X, v2.X))
		maxX := max(v0.X, max(v1.X, v2.X))
		for x := minX; x <= maxX; x++ {
			// Interpolate U,V across the line
			t := float64(x-minX) / float64(maxX-minX+1)
			u := clampUint8(int(float64(v0.U) + t*float64(int(v2.U)-int(v0.U))))
			v := clampUint8(int(float64(v0.V) + t*float64(int(v2.V)-int(v0.V))))
			r, cell := shader(u, v, x, v0.Y)
			cell.Rune = r
			c.SetCell(x, v0.Y, cell)
		}
		return
	}

	// Check for flat triangles
	if v1.Y == v2.Y {
		fillFlatBottomShaded(c, v0, v1, v2, shader)
	} else if v0.Y == v1.Y {
		fillFlatTopShaded(c, v0, v1, v2, shader)
	} else {
		// General case: split into flat-bottom and flat-top
		t := float64(v1.Y-v0.Y) / float64(v2.Y-v0.Y)
		v3 := Vertex2D{
			X: int(float64(v0.X) + t*float64(v2.X-v0.X) + 0.5),
			Y: v1.Y,
			U: clampUint8(int(float64(v0.U) + t*float64(int(v2.U)-int(v0.U)))),
			V: clampUint8(int(float64(v0.V) + t*float64(int(v2.V)-int(v0.V)))),
		}

		fillFlatBottomShaded(c, v0, v1, v3, shader)
		fillFlatTopShaded(c, v1, v3, v2, shader)
	}
}

func fillFlatBottomShaded(c *canvas.Canvas, v0, v1, v2 Vertex2D, shader PixelShader) {
	if v1.Y == v0.Y {
		return
	}

	invSlope1 := float64(v1.X-v0.X) / float64(v1.Y-v0.Y)
	invSlope2 := float64(v2.X-v0.X) / float64(v2.Y-v0.Y)

	curX1 := float64(v0.X)
	curX2 := float64(v0.X)

	for y := v0.Y; y <= v1.Y; y++ {
		t := float64(y-v0.Y) / float64(v1.Y-v0.Y)

		// Interpolate UV for left edge (v0 to v1)
		u1 := clampUint8(int(float64(v0.U) + t*float64(int(v1.U)-int(v0.U))))
		vv1 := clampUint8(int(float64(v0.V) + t*float64(int(v1.V)-int(v0.V))))

		// Interpolate UV for right edge (v0 to v2)
		u2 := clampUint8(int(float64(v0.U) + t*float64(int(v2.U)-int(v0.U))))
		vv2 := clampUint8(int(float64(v0.V) + t*float64(int(v2.V)-int(v0.V))))

		x1 := int(curX1 + 0.5)
		x2 := int(curX2 + 0.5)
		if x1 > x2 {
			x1, x2 = x2, x1
			u1, u2 = u2, u1
			vv1, vv2 = vv2, vv1
		}

		drawShadedScanline(c, x1, x2, y, u1, vv1, u2, vv2, shader)

		curX1 += invSlope1
		curX2 += invSlope2
	}
}

func fillFlatTopShaded(c *canvas.Canvas, v0, v1, v2 Vertex2D, shader PixelShader) {
	if v2.Y == v0.Y {
		return
	}

	invSlope1 := float64(v2.X-v0.X) / float64(v2.Y-v0.Y)
	invSlope2 := float64(v2.X-v1.X) / float64(v2.Y-v1.Y)

	curX1 := float64(v2.X)
	curX2 := float64(v2.X)

	for y := v2.Y; y >= v0.Y; y-- {
		t := float64(v2.Y-y) / float64(v2.Y-v0.Y)

		// Interpolate UV for left edge (v2 to v0)
		u1 := clampUint8(int(float64(v2.U) + t*float64(int(v0.U)-int(v2.U))))
		vv1 := clampUint8(int(float64(v2.V) + t*float64(int(v0.V)-int(v2.V))))

		// Interpolate UV for right edge (v2 to v1)
		u2 := clampUint8(int(float64(v2.U) + t*float64(int(v1.U)-int(v2.U))))
		vv2 := clampUint8(int(float64(v2.V) + t*float64(int(v1.V)-int(v2.V))))

		x1 := int(curX1 + 0.5)
		x2 := int(curX2 + 0.5)
		if x1 > x2 {
			x1, x2 = x2, x1
			u1, u2 = u2, u1
			vv1, vv2 = vv2, vv1
		}

		drawShadedScanline(c, x1, x2, y, u1, vv1, u2, vv2, shader)

		curX1 -= invSlope1
		curX2 -= invSlope2
	}
}

func drawShadedScanline(c *canvas.Canvas, x1, x2, y int, u1, v1, u2, v2 uint8, shader PixelShader) {
	if x1 == x2 {
		if c.InBounds(x1, y) {
			r, cell := shader(u1, v1, x1, y)
			cell.Rune = r
			c.SetCell(x1, y, cell)
		}
		return
	}

	dx := x2 - x1
	for x := x1; x <= x2; x++ {
		t := float64(x-x1) / float64(dx)
		u := clampUint8(int(float64(u1) + t*float64(int(u2)-int(u1))))
		v := clampUint8(int(float64(v1) + t*float64(int(v2)-int(v1))))

		if c.InBounds(x, y) {
			r, cell := shader(u, v, x, y)
			cell.Rune = r
			c.SetCell(x, y, cell)
		}
	}
}

// SimpleShader creates a pixel shader with a constant color and gradient-based character.
func SimpleShader(color canvas.Color, grad *Gradient) PixelShader {
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		// Use average of U and V as intensity
		intensity := (uint16(u) + uint16(v)) / 2
		char := grad.Char(uint8(intensity))
		return char, canvas.NewCell(char, color)
	}
}

// SolidShader creates a pixel shader with a constant character and color.
func SolidShader(r rune, color canvas.Color) PixelShader {
	cell := canvas.NewCell(r, color)
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		return r, cell
	}
}

// GradientShader creates a pixel shader that maps UV to gradient characters.
func GradientShader(grad *Gradient, color canvas.Color) PixelShader {
	return func(u, v uint8, x, y int) (rune, canvas.Cell) {
		char := grad.Char(u)
		return char, canvas.NewCell(char, color)
	}
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
