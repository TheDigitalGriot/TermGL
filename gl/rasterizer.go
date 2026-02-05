package gl

import (
	stdmath "math"

	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/math"
)

// ProjectedVertex represents a vertex after projection to screen space.
type ProjectedVertex struct {
	Position math.Vec3 // Screen-space position (X, Y in pixels, Z is depth)
	Normal   math.Vec3 // World-space normal for lighting
}

// ProjectedTriangle represents a triangle in screen space.
type ProjectedTriangle struct {
	Vertices   [3]ProjectedVertex
	FaceNormal math.Vec3
}

// ShadeFunc is called for each pixel during rasterization.
// It receives the screen coordinates and barycentric weights.
// Returns the character to draw at that position.
type ShadeFunc func(x, y int, w1, w2, w3 float64, depth float64) rune

// Rasterizer handles triangle rasterization to a canvas.
type Rasterizer struct {
	canvas *canvas.Canvas

	// Z-buffer plane equation cache for current triangle
	zCross math.Vec3
	zVert  math.Vec3

	// Original triangle vertices for barycentric interpolation
	origP1, origP2, origP3 math.Vec3
}

// NewRasterizer creates a new rasterizer for the given canvas.
func NewRasterizer(c *canvas.Canvas) *Rasterizer {
	return &Rasterizer{
		canvas: c,
	}
}

// RasterizeTriangle fills a triangle using scanline rasterization.
// This is ported from ascii-graphics-3d/src/Screen.cpp:222-324
func (r *Rasterizer) RasterizeTriangle(tri ProjectedTriangle, shadeFunc ShadeFunc) {
	// Store original vertices for barycentric calculation
	r.origP1 = tri.Vertices[0].Position
	r.origP2 = tri.Vertices[1].Position
	r.origP3 = tri.Vertices[2].Position

	// Calculate z-buffer plane equation
	v1 := tri.Vertices[1].Position.Sub(tri.Vertices[0].Position)
	v2 := tri.Vertices[2].Position.Sub(tri.Vertices[0].Position)
	r.zCross = v1.Cross(v2)
	r.zVert = tri.Vertices[0].Position

	// Sort vertices by Y coordinate (ascending)
	verts := tri.Vertices
	if verts[0].Position.Y > verts[1].Position.Y {
		verts[0], verts[1] = verts[1], verts[0]
	}
	if verts[1].Position.Y > verts[2].Position.Y {
		verts[1], verts[2] = verts[2], verts[1]
	}
	if verts[0].Position.Y > verts[1].Position.Y {
		verts[0], verts[1] = verts[1], verts[0]
	}

	// Check for flat triangles
	if verts[1].Position.Y == verts[2].Position.Y {
		// Flat bottom
		r.fillFlatBottom(verts[0], verts[1], verts[2], shadeFunc)
	} else if verts[0].Position.Y == verts[1].Position.Y {
		// Flat top
		r.fillFlatTop(verts[0], verts[1], verts[2], shadeFunc)
	} else {
		// General case: split into flat-bottom and flat-top triangles
		// Find the x-coordinate of the splitting point
		m1 := (verts[0].Position.Y - verts[2].Position.Y) /
			(verts[0].Position.X - verts[2].Position.X)

		var newX float64
		if stdmath.IsInf(m1, 0) {
			newX = verts[0].Position.X
		} else {
			b1 := verts[0].Position.Y - m1*verts[0].Position.X
			newX = (verts[1].Position.Y - b1) / m1
		}

		// Calculate Z at the new point using plane equation
		newZ := math.CalcZ(newX, verts[1].Position.Y, r.zCross, r.zVert)

		// Create the splitting vertex (interpolate normal too)
		newVert := ProjectedVertex{
			Position: math.Vec3{X: newX, Y: verts[1].Position.Y, Z: newZ},
			Normal:   verts[1].Normal, // Approximate
		}

		// Fill both sub-triangles
		r.fillFlatBottom(verts[0], newVert, verts[1], shadeFunc)
		r.fillFlatTop(newVert, verts[1], verts[2], shadeFunc)
	}
}

// fillFlatBottom fills a triangle where v1.Y == v2.Y (bottom edge is flat).
// v0 is the top vertex.
// Ported from ascii-graphics-3d/src/Screen.cpp:267-294
func (r *Rasterizer) fillFlatBottom(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFunc) {
	// Calculate inverse slopes
	var m1, b1, m2, b2 float64

	if v1.Position.X != v0.Position.X {
		m1 = (v0.Position.Y - v1.Position.Y) / (v0.Position.X - v1.Position.X)
		b1 = v0.Position.Y - m1*v0.Position.X
	}
	if v2.Position.X != v0.Position.X {
		m2 = (v0.Position.Y - v2.Position.Y) / (v0.Position.X - v2.Position.X)
		b2 = v0.Position.Y - m2*v0.Position.X
	}

	// Draw the top point
	r.drawPixel(int(stdmath.Round(v0.Position.X)), int(stdmath.Round(v0.Position.Y)), shadeFunc)

	// Fill scanlines from top to bottom
	startY := int(stdmath.Round(v0.Position.Y)) + 1
	endY := int(stdmath.Round(v1.Position.Y))

	for y := startY; y <= endY; y++ {
		var x1, x2 float64
		fy := float64(y)

		if v1.Position.X == v0.Position.X || stdmath.IsInf(m1, 0) {
			x1 = v0.Position.X
		} else {
			x1 = (fy - b1) / m1
		}

		if v2.Position.X == v0.Position.X || stdmath.IsInf(m2, 0) {
			x2 = v0.Position.X
		} else {
			x2 = (fy - b2) / m2
		}

		r.drawHLine(int(stdmath.Round(x1)), int(stdmath.Round(x2)), y, shadeFunc)
	}
}

// fillFlatTop fills a triangle where v0.Y == v1.Y (top edge is flat).
// v2 is the bottom vertex.
// Ported from ascii-graphics-3d/src/Screen.cpp:299-324
func (r *Rasterizer) fillFlatTop(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFunc) {
	// Calculate inverse slopes
	var m1, b1, m2, b2 float64

	if v2.Position.X != v0.Position.X {
		m1 = (v2.Position.Y - v0.Position.Y) / (v2.Position.X - v0.Position.X)
		b1 = v2.Position.Y - m1*v2.Position.X
	}
	if v2.Position.X != v1.Position.X {
		m2 = (v2.Position.Y - v1.Position.Y) / (v2.Position.X - v1.Position.X)
		b2 = v2.Position.Y - m2*v2.Position.X
	}

	// Draw the bottom point
	r.drawPixel(int(stdmath.Round(v2.Position.X)), int(stdmath.Round(v2.Position.Y)), shadeFunc)

	// Fill scanlines from bottom to top
	startY := int(stdmath.Round(v2.Position.Y))
	endY := int(stdmath.Round(v0.Position.Y))

	for y := startY; y >= endY; y-- {
		var x1, x2 float64
		fy := float64(y)

		if v2.Position.X == v0.Position.X || stdmath.IsInf(m1, 0) {
			x1 = v2.Position.X
		} else {
			x1 = (fy - b1) / m1
		}

		if v2.Position.X == v1.Position.X || stdmath.IsInf(m2, 0) {
			x2 = v2.Position.X
		} else {
			x2 = (fy - b2) / m2
		}

		r.drawHLine(int(stdmath.Round(x1)), int(stdmath.Round(x2)), y, shadeFunc)
	}
}

// drawHLine draws a horizontal scanline.
func (r *Rasterizer) drawHLine(x1, x2, y int, shadeFunc ShadeFunc) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		r.drawPixel(x, y, shadeFunc)
	}
}

// drawPixel draws a single pixel with z-buffer testing.
func (r *Rasterizer) drawPixel(x, y int, shadeFunc ShadeFunc) {
	if !r.canvas.InBounds(x, y) {
		return
	}

	// Calculate Z depth using plane equation
	depth := math.CalcZ(float64(x), float64(y), r.zCross, r.zVert)

	// Z-buffer test
	if !r.canvas.TestAndSetDepth(x, y, depth) {
		return
	}

	// Calculate barycentric coordinates for interpolation
	w1, w2, w3 := math.CalcBary(r.origP1, r.origP2, r.origP3, x, y)

	// Get the character from shade function
	char := shadeFunc(x, y, w1, w2, w3, depth)

	r.canvas.SetRune(x, y, char)
}

// RasterizeWireframe draws triangle edges only.
func (r *Rasterizer) RasterizeWireframe(tri ProjectedTriangle, char rune, c *canvas.Canvas) {
	// Setup z-buffer plane for depth testing
	v1 := tri.Vertices[1].Position.Sub(tri.Vertices[0].Position)
	v2 := tri.Vertices[2].Position.Sub(tri.Vertices[0].Position)
	r.zCross = v1.Cross(v2)
	r.zVert = tri.Vertices[0].Position

	cell := canvas.NewCell(char, canvas.White)

	// Draw three edges
	r.drawLineWithDepth(tri.Vertices[0].Position, tri.Vertices[1].Position, cell, c)
	r.drawLineWithDepth(tri.Vertices[1].Position, tri.Vertices[2].Position, cell, c)
	r.drawLineWithDepth(tri.Vertices[2].Position, tri.Vertices[0].Position, cell, c)
}

// drawLineWithDepth draws a line with z-buffer testing.
func (r *Rasterizer) drawLineWithDepth(p1, p2 math.Vec3, cell canvas.Cell, c *canvas.Canvas) {
	x1 := int(stdmath.Round(p1.X))
	y1 := int(stdmath.Round(p1.Y))
	x2 := int(stdmath.Round(p2.X))
	y2 := int(stdmath.Round(p2.Y))

	// Simple line drawing with depth interpolation
	dx := stdmath.Abs(float64(x2 - x1))
	dy := stdmath.Abs(float64(y2 - y1))
	steps := int(stdmath.Max(dx, dy))

	if steps == 0 {
		depth := math.CalcZ(float64(x1), float64(y1), r.zCross, r.zVert)
		if c.TestAndSetDepth(x1, y1, depth) {
			c.SetCell(x1, y1, cell)
		}
		return
	}

	xInc := float64(x2-x1) / float64(steps)
	yInc := float64(y2-y1) / float64(steps)

	x := float64(x1)
	y := float64(y1)

	for i := 0; i <= steps; i++ {
		ix := int(stdmath.Round(x))
		iy := int(stdmath.Round(y))

		if c.InBounds(ix, iy) {
			depth := math.CalcZ(x, y, r.zCross, r.zVert)
			if c.TestAndSetDepth(ix, iy, depth) {
				c.SetCell(ix, iy, cell)
			}
		}

		x += xInc
		y += yInc
	}
}
