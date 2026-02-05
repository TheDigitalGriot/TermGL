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

// ShadeFuncWithColor is called for each pixel during rasterization with color support.
// It receives the screen coordinates and barycentric weights.
// Returns the character and color to draw at that position.
type ShadeFuncWithColor func(x, y int, w1, w2, w3 float64, depth float64) (rune, canvas.Color)

// ============================================================================
// Screen-Space Vertex Types (matches TermGL-C-Plus TGLVert)
// ============================================================================

// ScreenVertex represents a vertex in screen space with UV coordinates.
// Matches C TGLVert structure from TermGL-C-Plus.
type ScreenVertex struct {
	X, Y int     // Screen coordinates (pixels)
	Z    float64 // Depth for z-buffer testing
	U, V uint8   // Texture coordinates (0-255)
}

// ScreenTriangle represents a triangle in screen space with UVs.
type ScreenTriangle struct {
	Vertices [3]ScreenVertex
}

// Rasterizer handles triangle rasterization to a canvas.
type Rasterizer struct {
	canvas *canvas.Canvas

	// Z-buffer plane equation cache for current triangle
	zCross math.Vec3
	zVert  math.Vec3

	// Original triangle vertices for barycentric interpolation
	origP1, origP2, origP3 math.Vec3

	// Original UVs for interpolation (used by shader-based rasterization)
	origUV1, origUV2, origUV3 [2]uint8
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
	// Wrap in color version with empty color
	r.RasterizeTriangleWithColor(tri, func(x, y int, w1, w2, w3 float64, depth float64) (rune, canvas.Color) {
		return shadeFunc(x, y, w1, w2, w3, depth), ""
	})
}

// RasterizeTriangleWithColor fills a triangle with color support.
func (r *Rasterizer) RasterizeTriangleWithColor(tri ProjectedTriangle, shadeFunc ShadeFuncWithColor) {
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
		r.fillFlatBottomWithColor(verts[0], verts[1], verts[2], shadeFunc)
	} else if verts[0].Position.Y == verts[1].Position.Y {
		// Flat top
		r.fillFlatTopWithColor(verts[0], verts[1], verts[2], shadeFunc)
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
		r.fillFlatBottomWithColor(verts[0], newVert, verts[1], shadeFunc)
		r.fillFlatTopWithColor(newVert, verts[1], verts[2], shadeFunc)
	}
}

// fillFlatBottom fills a triangle where v1.Y == v2.Y (bottom edge is flat).
// v0 is the top vertex.
// Ported from ascii-graphics-3d/src/Screen.cpp:267-294
func (r *Rasterizer) fillFlatBottom(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFunc) {
	r.fillFlatBottomWithColor(v0, v1, v2, func(x, y int, w1, w2, w3 float64, depth float64) (rune, canvas.Color) {
		return shadeFunc(x, y, w1, w2, w3, depth), ""
	})
}

func (r *Rasterizer) fillFlatBottomWithColor(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFuncWithColor) {
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
	r.drawPixelWithColor(int(stdmath.Round(v0.Position.X)), int(stdmath.Round(v0.Position.Y)), shadeFunc)

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

		r.drawHLineWithColor(int(stdmath.Round(x1)), int(stdmath.Round(x2)), y, shadeFunc)
	}
}

// fillFlatTop fills a triangle where v0.Y == v1.Y (top edge is flat).
// v2 is the bottom vertex.
// Ported from ascii-graphics-3d/src/Screen.cpp:299-324
func (r *Rasterizer) fillFlatTop(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFunc) {
	r.fillFlatTopWithColor(v0, v1, v2, func(x, y int, w1, w2, w3 float64, depth float64) (rune, canvas.Color) {
		return shadeFunc(x, y, w1, w2, w3, depth), ""
	})
}

func (r *Rasterizer) fillFlatTopWithColor(v0, v1, v2 ProjectedVertex, shadeFunc ShadeFuncWithColor) {
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
	r.drawPixelWithColor(int(stdmath.Round(v2.Position.X)), int(stdmath.Round(v2.Position.Y)), shadeFunc)

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

		r.drawHLineWithColor(int(stdmath.Round(x1)), int(stdmath.Round(x2)), y, shadeFunc)
	}
}

// drawHLine draws a horizontal scanline.
func (r *Rasterizer) drawHLine(x1, x2, y int, shadeFunc ShadeFunc) {
	r.drawHLineWithColor(x1, x2, y, func(x, y int, w1, w2, w3 float64, depth float64) (rune, canvas.Color) {
		return shadeFunc(x, y, w1, w2, w3, depth), ""
	})
}

// drawHLineWithColor draws a horizontal scanline with color support.
func (r *Rasterizer) drawHLineWithColor(x1, x2, y int, shadeFunc ShadeFuncWithColor) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		r.drawPixelWithColor(x, y, shadeFunc)
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

// drawPixelWithColor draws a single pixel with z-buffer testing and color support.
func (r *Rasterizer) drawPixelWithColor(x, y int, shadeFunc ShadeFuncWithColor) {
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

	// Get the character and color from shade function
	char, color := shadeFunc(x, y, w1, w2, w3, depth)

	if color != "" {
		r.canvas.SetCell(x, y, canvas.NewCell(char, color))
	} else {
		r.canvas.SetRune(x, y, char)
	}
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

// ============================================================================
// Shader-Based Rasterization (matches TermGL-C-Plus tgl_triangle_fill)
// ============================================================================

// RasterizeTriangleShader fills a triangle using a PixelShader.
// This is the main entry point for the new shader pipeline.
// UV coordinates are interpolated across the triangle and passed to the shader.
func (r *Rasterizer) RasterizeTriangleShader(tri ScreenTriangle, shader PixelShader, shaderData any) {
	// Store original UVs for interpolation
	r.origUV1 = [2]uint8{tri.Vertices[0].U, tri.Vertices[0].V}
	r.origUV2 = [2]uint8{tri.Vertices[1].U, tri.Vertices[1].V}
	r.origUV3 = [2]uint8{tri.Vertices[2].U, tri.Vertices[2].V}

	// Convert to float positions for math
	p1 := math.Vec3{X: float64(tri.Vertices[0].X), Y: float64(tri.Vertices[0].Y), Z: tri.Vertices[0].Z}
	p2 := math.Vec3{X: float64(tri.Vertices[1].X), Y: float64(tri.Vertices[1].Y), Z: tri.Vertices[1].Z}
	p3 := math.Vec3{X: float64(tri.Vertices[2].X), Y: float64(tri.Vertices[2].Y), Z: tri.Vertices[2].Z}

	// Store for barycentric calculation
	r.origP1 = p1
	r.origP2 = p2
	r.origP3 = p3

	// Calculate z-buffer plane equation
	v1 := p2.Sub(p1)
	v2 := p3.Sub(p1)
	r.zCross = v1.Cross(v2)
	r.zVert = p1

	// Sort vertices by Y coordinate (ascending)
	verts := tri.Vertices
	if verts[0].Y > verts[1].Y {
		verts[0], verts[1] = verts[1], verts[0]
	}
	if verts[1].Y > verts[2].Y {
		verts[1], verts[2] = verts[2], verts[1]
	}
	if verts[0].Y > verts[1].Y {
		verts[0], verts[1] = verts[1], verts[0]
	}

	// Check for flat triangles
	if verts[1].Y == verts[2].Y {
		// Flat bottom
		r.fillFlatBottomShader(verts[0], verts[1], verts[2], shader, shaderData)
	} else if verts[0].Y == verts[1].Y {
		// Flat top
		r.fillFlatTopShader(verts[0], verts[1], verts[2], shader, shaderData)
	} else {
		// General case: split into flat-bottom and flat-top triangles
		// Interpolate to find the split point
		t := float64(verts[1].Y-verts[0].Y) / float64(verts[2].Y-verts[0].Y)
		newX := int(float64(verts[0].X) + t*float64(verts[2].X-verts[0].X))
		newZ := verts[0].Z + t*(verts[2].Z-verts[0].Z)
		newU := uint8(float64(verts[0].U) + t*float64(int(verts[2].U)-int(verts[0].U)))
		newV := uint8(float64(verts[0].V) + t*float64(int(verts[2].V)-int(verts[0].V)))

		newVert := ScreenVertex{
			X: newX,
			Y: verts[1].Y,
			Z: newZ,
			U: newU,
			V: newV,
		}

		// Fill both sub-triangles
		r.fillFlatBottomShader(verts[0], newVert, verts[1], shader, shaderData)
		r.fillFlatTopShader(newVert, verts[1], verts[2], shader, shaderData)
	}
}

// fillFlatBottomShader fills a flat-bottom triangle with a pixel shader.
// v0 is the top vertex, v1 and v2 are the bottom vertices.
func (r *Rasterizer) fillFlatBottomShader(v0, v1, v2 ScreenVertex, shader PixelShader, shaderData any) {
	if v1.Y == v0.Y {
		return // Degenerate triangle
	}

	// Calculate slopes
	invSlope1 := float64(v1.X-v0.X) / float64(v1.Y-v0.Y)
	invSlope2 := float64(v2.X-v0.X) / float64(v2.Y-v0.Y)

	// Starting x positions at top
	curX1 := float64(v0.X)
	curX2 := float64(v0.X)

	// Calculate UV slopes
	uSlope1 := float64(int(v1.U)-int(v0.U)) / float64(v1.Y-v0.Y)
	vSlope1 := float64(int(v1.V)-int(v0.V)) / float64(v1.Y-v0.Y)
	uSlope2 := float64(int(v2.U)-int(v0.U)) / float64(v2.Y-v0.Y)
	vSlope2 := float64(int(v2.V)-int(v0.V)) / float64(v2.Y-v0.Y)

	curU1, curV1 := float64(v0.U), float64(v0.V)
	curU2, curV2 := float64(v0.U), float64(v0.V)

	// Z slopes
	zSlope1 := (v1.Z - v0.Z) / float64(v1.Y-v0.Y)
	zSlope2 := (v2.Z - v0.Z) / float64(v2.Y-v0.Y)
	curZ1, curZ2 := v0.Z, v0.Z

	for y := v0.Y; y <= v1.Y; y++ {
		r.drawHLineShader(int(curX1), int(curX2), y, curZ1, curZ2, curU1, curV1, curU2, curV2, shader, shaderData)

		curX1 += invSlope1
		curX2 += invSlope2
		curU1 += uSlope1
		curV1 += vSlope1
		curU2 += uSlope2
		curV2 += vSlope2
		curZ1 += zSlope1
		curZ2 += zSlope2
	}
}

// fillFlatTopShader fills a flat-top triangle with a pixel shader.
// v0 and v1 are the top vertices, v2 is the bottom vertex.
func (r *Rasterizer) fillFlatTopShader(v0, v1, v2 ScreenVertex, shader PixelShader, shaderData any) {
	if v2.Y == v0.Y {
		return // Degenerate triangle
	}

	// Calculate slopes
	invSlope1 := float64(v2.X-v0.X) / float64(v2.Y-v0.Y)
	invSlope2 := float64(v2.X-v1.X) / float64(v2.Y-v1.Y)

	// Starting x positions at bottom
	curX1 := float64(v2.X)
	curX2 := float64(v2.X)

	// Calculate UV slopes
	uSlope1 := float64(int(v0.U)-int(v2.U)) / float64(v0.Y-v2.Y)
	vSlope1 := float64(int(v0.V)-int(v2.V)) / float64(v0.Y-v2.Y)
	uSlope2 := float64(int(v1.U)-int(v2.U)) / float64(v1.Y-v2.Y)
	vSlope2 := float64(int(v1.V)-int(v2.V)) / float64(v1.Y-v2.Y)

	curU1, curV1 := float64(v2.U), float64(v2.V)
	curU2, curV2 := float64(v2.U), float64(v2.V)

	// Z slopes
	zSlope1 := (v0.Z - v2.Z) / float64(v0.Y-v2.Y)
	zSlope2 := (v1.Z - v2.Z) / float64(v1.Y-v2.Y)
	curZ1, curZ2 := v2.Z, v2.Z

	for y := v2.Y; y >= v0.Y; y-- {
		r.drawHLineShader(int(curX1), int(curX2), y, curZ1, curZ2, curU1, curV1, curU2, curV2, shader, shaderData)

		curX1 -= invSlope1
		curX2 -= invSlope2
		curU1 -= uSlope1
		curV1 -= vSlope1
		curU2 -= uSlope2
		curV2 -= vSlope2
		curZ1 -= zSlope1
		curZ2 -= zSlope2
	}
}

// drawHLineShader draws a horizontal scanline using a pixel shader.
// Interpolates z, u, v across the line matching C horiz_line function.
func (r *Rasterizer) drawHLineShader(x1, x2, y int, z1, z2, u1, v1, u2, v2 float64, shader PixelShader, shaderData any) {
	if x1 > x2 {
		x1, x2 = x2, x1
		z1, z2 = z2, z1
		u1, u2 = u2, u1
		v1, v2 = v2, v1
	}

	dx := x2 - x1
	if dx == 0 {
		r.drawPixelShader(x1, y, z1, uint8(u1), uint8(v1), shader, shaderData)
		return
	}

	for x := x1; x <= x2; x++ {
		// Linear interpolation along scanline
		t := float64(x-x1) / float64(dx)
		z := z1 + t*(z2-z1)
		u := u1 + t*(u2-u1)
		v := v1 + t*(v2-v1)

		r.drawPixelShader(x, y, z, uint8(u), uint8(v), shader, shaderData)
	}
}

// drawPixelShader draws a single pixel using a pixel shader.
// Performs z-buffer testing and calls the shader with interpolated UVs.
func (r *Rasterizer) drawPixelShader(x, y int, z float64, u, v uint8, shader PixelShader, shaderData any) {
	if !r.canvas.InBounds(x, y) {
		return
	}

	// Z-buffer test
	if !r.canvas.TestAndSetDepth(x, y, z) {
		return
	}

	// Call the pixel shader
	char, fg, bg := shader(u, v, shaderData)

	// Set the cell
	cell := canvas.Cell{Rune: char}
	if fg != "" {
		cell.Foreground = fg
		cell.HasFg = true
	}
	if bg != "" {
		cell.Background = bg
		cell.HasBg = true
	}
	r.canvas.SetCell(x, y, cell)
}
