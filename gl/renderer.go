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
	RenderTextured                    // UV texture mapping
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

	// Texture for RenderTextured mode
	Texture *Texture

	// Internal
	rasterizer *Rasterizer
	pipeline   *Pipeline3D

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
		pipeline:     NewPipeline3D(c),
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

			// Simple near-plane clipping check (very permissive to avoid over-clipping)
			// Only clip if vertex is behind the camera
			if clipPos.Z < -10 {
				allInFront = false
				break
			}

			// Viewport transform with terminal cell aspect ratio compensation
			// Flip Y (screen Y is down, world Y is up)
			// Terminal cells are ~2x taller than wide, so stretch X by AspectScale
			// The camera projection already accounts for screen aspect ratio
			screenX := clipPos.X*halfHeight*r.AspectScale + halfWidth
			screenY := -clipPos.Y*halfHeight + halfHeight

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
		case RenderTextured:
			r.renderTextured(projected, tri)
		}
	}
}

// renderShaded renders a triangle with lighting.
func (r *Renderer) renderShaded(tri ProjectedTriangle) {
	// Check if we have a color ramp for colored shading
	hasColorRamp := len(r.Shader.ColorRamp) > 0

	if r.ShadingMode == ShadingFlat {
		if hasColorRamp {
			// Flat shading with colors
			shadeChar, shadeColor := r.Shader.CalculateFlatShadeWithColor(tri.FaceNormal)
			r.rasterizer.RasterizeTriangleWithColor(tri, func(x, y int, w1, w2, w3, depth float64) (rune, canvas.Color) {
				return shadeChar, shadeColor
			})
		} else {
			// Flat shading: one shade for entire triangle
			shadeChar := r.Shader.CalculateFlatShade(tri.FaceNormal)
			r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
				return shadeChar
			})
		}
	} else {
		// Smooth shading: interpolate normals
		n1 := tri.Vertices[0].Normal
		n2 := tri.Vertices[1].Normal
		n3 := tri.Vertices[2].Normal

		if hasColorRamp {
			r.rasterizer.RasterizeTriangleWithColor(tri, func(x, y int, w1, w2, w3, depth float64) (rune, canvas.Color) {
				return r.Shader.CalculateSmoothShadeWithColor(n1, n2, n3, w1, w2, w3)
			})
		} else {
			r.rasterizer.RasterizeTriangle(tri, func(x, y int, w1, w2, w3, depth float64) rune {
				return r.Shader.CalculateSmoothShade(n1, n2, n3, w1, w2, w3)
			})
		}
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

// SetColorRamp sets the color gradient for shading.
// Colors should match the length of the luminance ramp.
func (r *Renderer) SetColorRamp(colors []canvas.Color) {
	r.Shader.SetColorRamp(colors)
}

// SetShadingRamp sets both the character and color ramps for shading.
func (r *Renderer) SetShadingRamp(ramp string, colors []canvas.Color) {
	r.Shader.SetLuminanceRamp(ramp)
	r.Shader.SetColorRamp(colors)
}

// SetDirectionalLight sets the directional light.
func (r *Renderer) SetDirectionalLight(light *DirectionalLight) {
	r.Shader.DirectionalLight = light
}

// SetAmbientLight sets the ambient light.
func (r *Renderer) SetAmbientLight(light *AmbientLight) {
	r.Shader.AmbientLight = light
}

// SetTexture sets the texture for RenderTextured mode.
func (r *Renderer) SetTexture(tex *Texture) {
	r.Texture = tex
}

// renderTextured renders a triangle with texture mapping.
func (r *Renderer) renderTextured(projected ProjectedTriangle, original Triangle) {
	if r.Texture == nil {
		// Fall back to solid rendering if no texture
		r.renderSolid(projected, '#')
		return
	}

	// Convert projected triangle to screen triangle with UVs
	screenTri := ScreenTriangle{
		Vertices: [3]ScreenVertex{
			{
				X: int(stdmath.Round(projected.Vertices[0].Position.X)),
				Y: int(stdmath.Round(projected.Vertices[0].Position.Y)),
				Z: projected.Vertices[0].Position.Z,
				U: uint8(original.Vertices[0].UV.X * 255),
				V: uint8(original.Vertices[0].UV.Y * 255),
			},
			{
				X: int(stdmath.Round(projected.Vertices[1].Position.X)),
				Y: int(stdmath.Round(projected.Vertices[1].Position.Y)),
				Z: projected.Vertices[1].Position.Z,
				U: uint8(original.Vertices[1].UV.X * 255),
				V: uint8(original.Vertices[1].UV.Y * 255),
			},
			{
				X: int(stdmath.Round(projected.Vertices[2].Position.X)),
				Y: int(stdmath.Round(projected.Vertices[2].Position.Y)),
				Z: projected.Vertices[2].Position.Z,
				U: uint8(original.Vertices[2].UV.X * 255),
				V: uint8(original.Vertices[2].UV.Y * 255),
			},
		},
	}

	// Create pixel shader data
	pixData := &PixelShaderTexture{Texture: r.Texture}

	// Rasterize with texture shader
	r.rasterizer.RasterizeTriangleShader(screenTri, PixelShaderTextureFunc, pixData)
}

// ============================================================================
// Shader Pipeline API (matches TermGL-C-Plus architecture)
// ============================================================================

// RenderMeshShaded renders a mesh using custom vertex and pixel shaders.
// This provides full control over the rendering pipeline.
func (r *Renderer) RenderMeshShaded(
	mesh *Mesh,
	vertShader VertexShader,
	vertData any,
	pixShader PixelShader,
	pixData any,
) {
	// Update pipeline dimensions
	r.pipeline.Width = r.Canvas.Width()
	r.pipeline.Height = r.Canvas.Height()
	r.pipeline.CullFace = r.BackfaceCull

	// Render through the pipeline
	r.pipeline.DrawMesh3D(mesh, vertShader, vertData, pixShader, pixData)
}

// RenderMeshTextured renders a mesh with a texture using the shader pipeline.
// This is a convenience method for textured rendering.
func (r *Renderer) RenderMeshTextured(mesh *Mesh, tex *Texture) {
	// Build MVP matrix
	modelMatrix := mesh.Transform.Matrix()
	viewProjMatrix := r.Camera.ViewProjectionMatrix()
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	// Setup vertex shader
	vertData := &VertexShaderSimple{Mat: mvpMatrix}

	// Setup pixel shader
	pixData := &PixelShaderTexture{Texture: tex}

	// Render
	r.RenderMeshShaded(mesh, VertexShaderSimpleFunc, vertData, PixelShaderTextureFunc, pixData)
}

// GetPipeline returns the underlying 3D pipeline for advanced usage.
func (r *Renderer) GetPipeline() *Pipeline3D {
	return r.pipeline
}

// RenderMeshLit renders a mesh with lighting using the shader pipeline.
// Uses the new pipeline for proper frustum clipping while supporting
// the legacy shading system (flat/smooth with gradients).
func (r *Renderer) RenderMeshLit(mesh *Mesh) {
	// Build MVP matrix
	modelMatrix := mesh.Transform.Matrix()
	viewProjMatrix := r.Camera.ViewProjectionMatrix()
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	// Setup vertex shader
	vertData := &VertexShaderSimple{Mat: mvpMatrix}

	// Update pipeline dimensions and settings
	r.pipeline.Width = r.Canvas.Width()
	r.pipeline.Height = r.Canvas.Height()
	r.pipeline.CullFace = r.BackfaceCull

	for _, tri := range mesh.Triangles {
		// Transform normals to world space
		worldNormals := [3]math.Vec3{
			modelMatrix.MulVec3Dir(tri.Vertices[0].Normal).Normalize(),
			modelMatrix.MulVec3Dir(tri.Vertices[1].Normal).Normalize(),
			modelMatrix.MulVec3Dir(tri.Vertices[2].Normal).Normalize(),
		}

		// For lighting, we encode barycentric weights as UV coordinates
		// Vertex 0: UV=(255,0) → w1=1, w2=0, w3=0
		// Vertex 1: UV=(0,255) → w1=0, w2=1, w3=0
		// Vertex 2: UV=(0,0)   → w1=0, w2=0, w3=1
		barycentricUVs := [3][2]uint8{
			{255, 0}, // Vertex 0: full w1 weight
			{0, 255}, // Vertex 1: full w2 weight
			{0, 0},   // Vertex 2: full w3 weight
		}

		// Setup pixel shader with lighting data
		pixData := &PixelShaderLighting{
			Shader:  r.Shader,
			Normals: worldNormals,
		}

		// Get vertex positions
		positions := [3]math.Vec3{
			tri.Vertices[0].Position,
			tri.Vertices[1].Position,
			tri.Vertices[2].Position,
		}

		r.pipeline.DrawTriangle3D(
			positions,
			barycentricUVs,
			true, // filled
			VertexShaderSimpleFunc,
			vertData,
			PixelShaderLightingFunc,
			pixData,
		)
	}
}

// RenderMeshWireframe renders a mesh as wireframe using the shader pipeline.
func (r *Renderer) RenderMeshWireframe(mesh *Mesh, char rune, color canvas.Color) {
	// Build MVP matrix
	modelMatrix := mesh.Transform.Matrix()
	viewProjMatrix := r.Camera.ViewProjectionMatrix()
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	// Setup vertex shader
	vertData := &VertexShaderSimple{Mat: mvpMatrix}

	// Setup pixel shader with solid color
	pixData := &PixelShaderSolid{Char: char, Color: color}

	// Update pipeline dimensions and settings
	r.pipeline.Width = r.Canvas.Width()
	r.pipeline.Height = r.Canvas.Height()
	r.pipeline.CullFace = r.BackfaceCull

	for _, tri := range mesh.Triangles {
		// Get vertex positions
		positions := [3]math.Vec3{
			tri.Vertices[0].Position,
			tri.Vertices[1].Position,
			tri.Vertices[2].Position,
		}

		// Default UVs (not used for wireframe solid color)
		uvs := [3][2]uint8{{0, 0}, {0, 0}, {0, 0}}

		r.pipeline.DrawTriangle3D(
			positions,
			uvs,
			false, // wireframe
			VertexShaderSimpleFunc,
			vertData,
			PixelShaderSolidFunc,
			pixData,
		)
	}
}

// RenderMeshSolid renders a mesh with a solid character using the shader pipeline.
func (r *Renderer) RenderMeshSolid(mesh *Mesh, char rune, color canvas.Color) {
	// Build MVP matrix
	modelMatrix := mesh.Transform.Matrix()
	viewProjMatrix := r.Camera.ViewProjectionMatrix()
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	// Setup vertex shader
	vertData := &VertexShaderSimple{Mat: mvpMatrix}

	// Setup pixel shader with solid color
	pixData := &PixelShaderSolid{Char: char, Color: color}

	// Update pipeline dimensions and settings
	r.pipeline.Width = r.Canvas.Width()
	r.pipeline.Height = r.Canvas.Height()
	r.pipeline.CullFace = r.BackfaceCull

	for _, tri := range mesh.Triangles {
		// Get vertex positions
		positions := [3]math.Vec3{
			tri.Vertices[0].Position,
			tri.Vertices[1].Position,
			tri.Vertices[2].Position,
		}

		// Default UVs (not used for solid color)
		uvs := [3][2]uint8{{0, 0}, {0, 0}, {0, 0}}

		r.pipeline.DrawTriangle3D(
			positions,
			uvs,
			true, // filled
			VertexShaderSimpleFunc,
			vertData,
			PixelShaderSolidFunc,
			pixData,
		)
	}
}
