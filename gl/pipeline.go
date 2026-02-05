// Package gl provides 3D rendering pipeline for terminal graphics.
package gl

import (
	stdmath "math"

	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/math"
)

// Pipeline3D implements the complete 3D rendering pipeline.
// This matches the architecture of C tgl_triangle_3d from TermGL-C-Plus.
//
// Pipeline stages:
//  1. Vertex Shader - Transform object space to clip space
//  2. Backface Culling - Discard back-facing triangles
//  3. Frustum Clipping - Clip against all 6 frustum planes
//  4. Perspective Divide - Convert to normalized device coordinates
//  5. Screen Mapping - Convert NDC to screen pixels
//  6. Rasterization - Fill triangles with pixel shader
type Pipeline3D struct {
	Canvas     *canvas.Canvas
	Rasterizer *Rasterizer

	// Settings
	CullFace bool // Enable backface culling (default: true)
	ZBuffer  bool // Enable depth testing (default: true)

	// Dimensions
	Width  int
	Height int
}

// NewPipeline3D creates a new 3D rendering pipeline.
func NewPipeline3D(c *canvas.Canvas) *Pipeline3D {
	return &Pipeline3D{
		Canvas:     c,
		Rasterizer: NewRasterizer(c),
		CullFace:   true,
		ZBuffer:    true,
		Width:      c.Width(),
		Height:     c.Height(),
	}
}

// DrawTriangle3D renders a 3D triangle through the full pipeline.
// This is the main entry point matching C tgl_triangle_3d.
//
// Parameters:
//   - tri: Object-space triangle vertices
//   - uv: UV coordinates for each vertex (0-255 range)
//   - fill: true for filled triangle, false for wireframe
//   - vertShader: Vertex shader function
//   - vertData: Data passed to vertex shader
//   - pixShader: Pixel shader function
//   - pixData: Data passed to pixel shader
func (p *Pipeline3D) DrawTriangle3D(
	tri [3]math.Vec3,
	uv [3][2]uint8,
	fill bool,
	vertShader VertexShader,
	vertData any,
	pixShader PixelShader,
	pixData any,
) {
	// Stage 1: Vertex Shader - Transform to clip space
	var clipVerts [3]math.Vec4
	for i := 0; i < 3; i++ {
		clipVerts[i] = vertShader(tri[i], vertData)
	}

	// Stage 2: Backface Culling
	if p.CullFace {
		// Perspective divide for culling test
		var ndc [3]math.Vec3
		for i := 0; i < 3; i++ {
			w := clipVerts[i].W
			if w != 0 {
				ndc[i] = math.Vec3{
					X: clipVerts[i].X / w,
					Y: clipVerts[i].Y / w,
					Z: clipVerts[i].Z / w,
				}
			}
		}

		// Cross product of edges to get winding
		ab := ndc[1].Sub(ndc[0])
		ac := ndc[2].Sub(ndc[0])
		cross := ab.Cross(ac)

		// If z component is negative, triangle faces away from camera
		if cross.Z < 0 {
			return
		}
	}

	// Stage 3: Frustum Clipping
	clipTri := ClipTriangle{
		Vertices: [3]ClipVertex{
			{Position: clipVerts[0], UV: uv[0]},
			{Position: clipVerts[1], UV: uv[1]},
			{Position: clipVerts[2], UV: uv[2]},
		},
	}

	clippedTriangles := ClipTriangleAgainstFrustum(clipTri)
	if len(clippedTriangles) == 0 {
		return
	}

	// Process each clipped triangle
	halfWidth := float64(p.Width) * 0.5
	halfHeight := float64(p.Height) * 0.5

	for _, ct := range clippedTriangles {
		// Stage 4 & 5: Perspective Divide and Screen Mapping
		var screenTri ScreenTriangle
		for i := 0; i < 3; i++ {
			v := ct.Vertices[i]
			w := v.Position.W
			if w == 0 {
				w = 1 // Avoid division by zero
			}

			// Perspective divide
			ndcX := v.Position.X / w
			ndcY := v.Position.Y / w
			ndcZ := v.Position.Z / w

			// Screen mapping: NDC [-1, 1] to screen [0, width/height]
			screenX := (ndcX + 1) * halfWidth
			screenY := (1 - ndcY) * halfHeight // Flip Y for screen coordinates

			screenTri.Vertices[i] = ScreenVertex{
				X: int(stdmath.Round(screenX)),
				Y: int(stdmath.Round(screenY)),
				Z: ndcZ,
				U: v.UV[0],
				V: v.UV[1],
			}
		}

		// Stage 6: Rasterization
		if fill {
			p.Rasterizer.RasterizeTriangleShader(screenTri, pixShader, pixData)
		} else {
			// Wireframe mode
			p.drawWireframeShader(screenTri, pixShader, pixData)
		}
	}
}

// drawWireframeShader draws a triangle outline using a pixel shader.
func (p *Pipeline3D) drawWireframeShader(tri ScreenTriangle, shader PixelShader, shaderData any) {
	for i := 0; i < 3; i++ {
		v1 := tri.Vertices[i]
		v2 := tri.Vertices[(i+1)%3]
		p.drawLineShader(v1, v2, shader, shaderData)
	}
}

// drawLineShader draws a line between two screen vertices using a pixel shader.
func (p *Pipeline3D) drawLineShader(v1, v2 ScreenVertex, shader PixelShader, shaderData any) {
	dx := stdmath.Abs(float64(v2.X - v1.X))
	dy := stdmath.Abs(float64(v2.Y - v1.Y))
	steps := int(stdmath.Max(dx, dy))

	if steps == 0 {
		p.Rasterizer.drawPixelShader(v1.X, v1.Y, v1.Z, v1.U, v1.V, shader, shaderData)
		return
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(float64(v1.X) + t*float64(v2.X-v1.X))
		y := int(float64(v1.Y) + t*float64(v2.Y-v1.Y))
		z := v1.Z + t*(v2.Z-v1.Z)
		u := uint8(float64(v1.U) + t*float64(int(v2.U)-int(v1.U)))
		v := uint8(float64(v1.V) + t*float64(int(v2.V)-int(v1.V)))

		p.Rasterizer.drawPixelShader(x, y, z, u, v, shader, shaderData)
	}
}

// DrawMesh3D renders an entire mesh through the pipeline.
func (p *Pipeline3D) DrawMesh3D(
	mesh *Mesh,
	vertShader VertexShader,
	vertData any,
	pixShader PixelShader,
	pixData any,
) {
	for _, tri := range mesh.Triangles {
		// Extract positions and UVs
		positions := [3]math.Vec3{
			tri.Vertices[0].Position,
			tri.Vertices[1].Position,
			tri.Vertices[2].Position,
		}

		uvs := [3][2]uint8{
			{uint8(tri.Vertices[0].UV.X * 255), uint8(tri.Vertices[0].UV.Y * 255)},
			{uint8(tri.Vertices[1].UV.X * 255), uint8(tri.Vertices[1].UV.Y * 255)},
			{uint8(tri.Vertices[2].UV.X * 255), uint8(tri.Vertices[2].UV.Y * 255)},
		}

		p.DrawTriangle3D(positions, uvs, true, vertShader, vertData, pixShader, pixData)
	}
}

// Clear clears the canvas and resets the z-buffer.
func (p *Pipeline3D) Clear() {
	p.Canvas.Clear()
}

// Resize updates the pipeline dimensions.
func (p *Pipeline3D) Resize(width, height int) {
	p.Width = width
	p.Height = height
}
