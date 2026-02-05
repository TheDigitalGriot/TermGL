package gl

import (
	stdmath "math"

	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/math"
)

// RenderMode determines how meshes are rendered.
type RenderMode int

const (
	RenderShaded    RenderMode = iota // Lit with character shading
	RenderWireframe                   // Edge lines only
	RenderSolid                       // Single character, no lighting
	RenderOutlined                    // Wireframe + solid
	RenderDepth                       // Z-buffer visualization
)

// ShadingMode determines flat vs smooth shading.
type ShadingMode int

const (
	ShadingFlat   ShadingMode = iota // One shade per face
	ShadingSmooth                    // Interpolated vertex normals (Gouraud)
)

// Renderer handles 3D rendering to a canvas.
type Renderer struct {
	Canvas *canvas.Canvas
	Camera *Camera

	// Rendering settings
	RenderMode   RenderMode
	ShadingMode  ShadingMode
	BackfaceCull bool

	// Shading
	Shader *Shader

	// Internal
	rasterizer *Rasterizer

	// Viewport transform scale (compensates for terminal cell aspect ratio)
	// Terminal cells are typically ~2:1 (height:width), so we scale X
	AspectScale float64
}

// NewRenderer creates a renderer for the given canvas and camera.
func NewRenderer(c *canvas.Canvas, cam *Camera) *Renderer {
	return &Renderer{
		Canvas:       c,
		Camera:       cam,
		RenderMode:   RenderShaded,
		ShadingMode:  ShadingFlat,
		BackfaceCull: true,
		Shader:       NewShader(),
		rasterizer:   NewRasterizer(c),
		AspectScale:  2.0, // Terminal cells are ~2x taller than wide
	}
}

// Clear clears the canvas and z-buffer.
func (r *Renderer) Clear() {
	r.Canvas.Clear()
}

// RenderMesh renders a mesh to the canvas.
func (r *Renderer) RenderMesh(mesh *Mesh) {
	// Get the model matrix from mesh transform
	modelMatrix := mesh.Transform.Matrix()

	// Get view-projection matrix
	viewProjMatrix := r.Camera.ViewProjectionMatrix()

	// Combined MVP matrix
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	// Screen dimensions
	width := float64(r.Canvas.Width())
	height := float64(r.Canvas.Height())
	halfWidth := width / 2
	halfHeight := height / 2

	// Viewport scale factors
	// X gets scaled by aspect ratio to fill width, then by AspectScale for cell shape
	// Y gets scaled by height
	// AspectScale compensates for terminal cells being taller than wide
	aspect := width / height
	xScale := aspect * r.AspectScale
	yScale := 1.0 // Can be tuned if needed

	for _, tri := range mesh.Triangles {
		// Transform face normal to world space for backface culling
		worldNormal := modelMatrix.MulVec3Dir(tri.FaceNormal).Normalize()

		// Get a vertex position in world space for backface test
		worldPos := modelMatrix.MulVec3(tri.Vertices[0].Position)

		// Backface culling: check if triangle faces away from camera
		if r.BackfaceCull {
			viewDir := worldPos.Sub(r.Camera.Position)
			if worldNormal.Dot(viewDir) >= 0 {
				continue // Face is pointing away from camera
			}
		}

		// Project vertices to screen space
		var projected ProjectedTriangle
		projected.FaceNormal = worldNormal

		allInFront := true
		for i, vert := range tri.Vertices {
			// Transform to clip space
			clipPos := mvpMatrix.MulVec3(vert.Position)

			// Simple near-plane clipping check
			if clipPos.Z < -1 {
				allInFront = false
				break
			}

			// Viewport transform with aspect ratio compensation
			// Flip Y (screen Y is down, world Y is up)
			// X: scaled by aspect * AspectScale, then offset to center
			// Y: scaled by halfHeight * yScale, then offset to center
			screenX := clipPos.X*xScale + halfWidth
			screenY := -clipPos.Y*halfHeight*yScale + halfHeight

			projected.Vertices[i] = ProjectedVertex{
				Position: math.Vec3{X: screenX, Y: screenY, Z: clipPos.Z},
				Normal:   modelMatrix.MulVec3Dir(vert.Normal).Normalize(),
			}
		}

		if !allInFront {
			continue
		}

		// Render based on mode
		switch r.RenderMode {
		case RenderShaded:
			r.renderShaded(projected)
		case RenderWireframe:
			r.renderWireframe(projected)
		case RenderSolid:
			r.renderSolid(projected, '#')
		case RenderOutlined:
			r.renderSolid(projected, '#')
			r.renderWireframe(projected)
		case RenderDepth:
			r.renderDepth(projected)
		}
	}
}

// renderShaded renders a triangle with lighting.
func (r *Renderer) renderShaded(tri ProjectedTriangle) {
	if r.ShadingMode == ShadingFlat {
		// Flat shading: one shade for entire triangle
		shadeChar := r.Shader.CalculateFlatShade(tri.FaceNormal)

		r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
			return shadeChar
		})
	} else {
		// Smooth shading: interpolate normals
		n1 := tri.Vertices[0].Normal
		n2 := tri.Vertices[1].Normal
		n3 := tri.Vertices[2].Normal

		r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
			return r.Shader.CalculateSmoothShade(n1, n2, n3, w1, w2, w3)
		})
	}
}

// renderWireframe renders triangle edges only.
func (r *Renderer) renderWireframe(tri ProjectedTriangle) {
	r.rasterizer.RasterizeWireframe(tri, '#', r.Canvas)
}

// renderSolid renders a triangle with a single character.
func (r *Renderer) renderSolid(tri ProjectedTriangle, char rune) {
	r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
		return char
	})
}

// renderDepth renders the z-buffer as a grayscale visualization.
func (r *Renderer) renderDepth(tri ProjectedTriangle) {
	r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
		// Map depth to luminance ramp
		// Normalize depth (assuming -1 to 1 range from NDC)
		normalized := (depth + 1) / 2
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		// Invert so closer = brighter
		intensity := 1 - normalized

		rampLen := len(r.Shader.LuminanceRamp)
		index := int(stdmath.Round(intensity * float64(rampLen-1)))
		if index < 0 {
			index = 0
		}
		if index >= rampLen {
			index = rampLen - 1
		}
		return rune(r.Shader.LuminanceRamp[index])
	})
}

// SetLuminanceRamp sets the character ramp for shading.
func (r *Renderer) SetLuminanceRamp(ramp string) {
	r.Shader.SetLuminanceRamp(ramp)
}

// SetDirectionalLight sets the directional light.
func (r *Renderer) SetDirectionalLight(light *DirectionalLight) {
	r.Shader.DirectionalLight = light
}

// SetAmbientLight sets the ambient light.
func (r *Renderer) SetAmbientLight(light *AmbientLight) {
	r.Shader.AmbientLight = light
}
