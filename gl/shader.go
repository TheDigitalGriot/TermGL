package gl

import (
	stdmath "math"

	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/math"
)

// ============================================================================
// Programmable Shader Types (matches TermGL-C-Plus architecture)
// ============================================================================

// VertexShader transforms a vertex from object space to clip space.
// Matches C TGLVertexShader: void (*)(const TGLVec3 in, TGLVec4 out, const void *data)
type VertexShader func(in math.Vec3, data any) math.Vec4

// PixelShader determines the output character and color for a pixel.
// Receives interpolated UV coordinates (0-255).
// Matches C TGLPixelShader: void (*)(uint8_t u, uint8_t v, TGLPixFmt *color, char *c, const void *data)
type PixelShader func(u, v uint8, data any) (char rune, fg canvas.Color, bg canvas.Color)

// ============================================================================
// Vertex Shader Data Structures
// ============================================================================

// VertexShaderSimple provides MVP matrix transformation.
// Matches C TGLVertexShaderSimple.
type VertexShaderSimple struct {
	Mat math.Mat4 // Combined Model-View-Projection matrix
}

// VertexShaderSimpleFunc is the built-in simple vertex shader.
// Transforms vertex position by the MVP matrix.
// Matches C tgl_vertex_shader_simple.
func VertexShaderSimpleFunc(in math.Vec3, data any) math.Vec4 {
	simple := data.(*VertexShaderSimple)
	return simple.Mat.MulVec4(math.Vec4{X: in.X, Y: in.Y, Z: in.Z, W: 1.0})
}

// ============================================================================
// Pixel Shader Data Structures
// ============================================================================

// PixelShaderSimple provides gradient + fixed color shading.
// Matches C TGLPixelShaderSimple.
type PixelShaderSimple struct {
	Color    canvas.Color
	Gradient *Gradient
}

// PixelShaderSimpleFunc maps u+v intensity to gradient character.
// Matches C tgl_pixel_shader_simple.
func PixelShaderSimpleFunc(u, v uint8, data any) (rune, canvas.Color, canvas.Color) {
	simple := data.(*PixelShaderSimple)
	// Average u and v for intensity (matching C behavior: u + v mapped to gradient)
	intensity := uint8((int(u) + int(v)) / 2)
	char := simple.Gradient.Char(intensity)
	return char, simple.Color, ""
}

// PixelShaderTexture provides texture mapping.
// Matches C TGLPixelShaderTexture.
type PixelShaderTexture struct {
	Texture *Texture
}

// PixelShaderTextureFunc samples a texture at the interpolated UV.
// Matches C tgl_pixel_shader_texture.
func PixelShaderTextureFunc(u, v uint8, data any) (rune, canvas.Color, canvas.Color) {
	tex := data.(*PixelShaderTexture)
	char, cell := tex.Texture.SampleNearest(u, v)
	return char, cell.Foreground, cell.Background
}

// PixelShaderLighting provides diffuse lighting based on normal interpolation.
// This bridges the old Shader lighting system to the new pixel shader API.
type PixelShaderLighting struct {
	Shader   *Shader   // The lighting shader to use
	Normals  [3]math.Vec3 // Triangle vertex normals for interpolation
}

// PixelShaderLightingFunc computes lighting based on interpolated normals.
// UV coordinates are interpreted as barycentric weights encoded as:
// u = w1*255, v = w2*255, w3 = 1 - w1 - w2 (implied)
func PixelShaderLightingFunc(u, v uint8, data any) (rune, canvas.Color, canvas.Color) {
	light := data.(*PixelShaderLighting)
	// Decode barycentric weights from UV
	w1 := float64(u) / 255.0
	w2 := float64(v) / 255.0
	w3 := 1.0 - w1 - w2
	if w3 < 0 {
		w3 = 0
	}
	char, color := light.Shader.CalculateSmoothShadeWithColor(
		light.Normals[0], light.Normals[1], light.Normals[2],
		w1, w2, w3,
	)
	return char, color, ""
}

// PixelShaderSolid provides a solid character and color.
type PixelShaderSolid struct {
	Char  rune
	Color canvas.Color
}

// PixelShaderSolidFunc returns a constant character and color.
func PixelShaderSolidFunc(u, v uint8, data any) (rune, canvas.Color, canvas.Color) {
	solid := data.(*PixelShaderSolid)
	return solid.Char, solid.Color, ""
}

// PixelShaderDepth visualizes depth as a gradient.
type PixelShaderDepth struct {
	Gradient *Gradient
	MinDepth float64
	MaxDepth float64
}

// PixelShaderDepthFunc maps depth to a gradient character.
// Uses V coordinate as depth (0=near, 255=far).
func PixelShaderDepthFunc(u, v uint8, data any) (rune, canvas.Color, canvas.Color) {
	depth := data.(*PixelShaderDepth)
	// V represents depth, invert so closer = brighter
	intensity := 255 - v
	char := depth.Gradient.Char(intensity)
	return char, canvas.White, ""
}

// ============================================================================
// Legacy Shader System (preserved for backwards compatibility)
// ============================================================================

// DefaultLuminanceRamp is the default character gradient for shading.
// Characters go from darkest (first) to brightest (last).
// Ported from ascii-graphics-3d: ".:+*=#%@"
const DefaultLuminanceRamp = " .:-=+*#%@"

// Shader calculates the visual appearance of surfaces.
type Shader struct {
	LuminanceRamp string
	ColorRamp     []canvas.Color // Optional: per-character colors (same length as LuminanceRamp)

	// Lighting
	DirectionalLight *DirectionalLight
	AmbientLight     *AmbientLight
}

// NewShader creates a new shader with default settings.
func NewShader() *Shader {
	return &Shader{
		LuminanceRamp:    DefaultLuminanceRamp,
		DirectionalLight: DefaultDirectionalLight(),
		AmbientLight:     NewAmbientLight(0.1),
	}
}

// SetLuminanceRamp sets the character ramp for shading.
func (s *Shader) SetLuminanceRamp(ramp string) {
	if len(ramp) > 0 {
		s.LuminanceRamp = ramp
	}
}

// CalculateFlatShade computes the shade character for flat shading.
// normal is the face normal, lightDir is the light direction.
//
// Ported from ascii-graphics-3d/src/Screen.cpp:343-352
func (s *Shader) CalculateFlatShade(normal math.Vec3) rune {
	r, _ := s.CalculateFlatShadeWithColor(normal)
	return r
}

// CalculateFlatShadeWithColor computes the shade character and color for flat shading.
func (s *Shader) CalculateFlatShadeWithColor(normal math.Vec3) (rune, canvas.Color) {
	// Calculate diffuse intensity using dot product
	diffuse := stdmath.Abs(normal.Dot(s.DirectionalLight.Direction))
	diffuse *= s.DirectionalLight.Intensity

	// Add ambient
	intensity := diffuse
	if s.AmbientLight != nil {
		intensity = intensity*(1-s.AmbientLight.Intensity) + s.AmbientLight.Intensity
	}

	return s.intensityToRuneAndColor(intensity)
}

// CalculateSmoothShade computes the shade for smooth (Gouraud) shading.
// It interpolates vertex normals using barycentric coordinates.
//
// Ported from ascii-graphics-3d/src/Screen.cpp:72-96
func (s *Shader) CalculateSmoothShade(n1, n2, n3 math.Vec3, w1, w2, w3 float64) rune {
	r, _ := s.CalculateSmoothShadeWithColor(n1, n2, n3, w1, w2, w3)
	return r
}

// CalculateSmoothShadeWithColor computes the shade character and color for smooth shading.
func (s *Shader) CalculateSmoothShadeWithColor(n1, n2, n3 math.Vec3, w1, w2, w3 float64) (rune, canvas.Color) {
	// Interpolate normal using barycentric weights
	nx := w1*n1.X + w2*n2.X + w3*n3.X
	ny := w1*n1.Y + w2*n2.Y + w3*n3.Y
	nz := w1*n1.Z + w2*n2.Z + w3*n3.Z

	// Normalize the interpolated normal
	length := stdmath.Sqrt(nx*nx + ny*ny + nz*nz)
	if length > 0 {
		nx /= length
		ny /= length
		nz /= length
	}

	pixelNormal := math.Vec3{X: nx, Y: ny, Z: nz}

	// Calculate diffuse intensity
	diffuse := stdmath.Abs(pixelNormal.Dot(s.DirectionalLight.Direction))
	diffuse *= s.DirectionalLight.Intensity

	// Add ambient
	intensity := diffuse
	if s.AmbientLight != nil {
		intensity = intensity*(1-s.AmbientLight.Intensity) + s.AmbientLight.Intensity
	}

	return s.intensityToRuneAndColor(intensity)
}

// intensityToRune maps an intensity value (0.0-1.0) to a character.
func (s *Shader) intensityToRune(intensity float64) rune {
	r, _ := s.intensityToRuneAndColor(intensity)
	return r
}

// intensityToRuneAndColor maps an intensity value (0.0-1.0) to a character and optional color.
func (s *Shader) intensityToRuneAndColor(intensity float64) (rune, canvas.Color) {
	// Clamp intensity
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}

	rampLen := len(s.LuminanceRamp)
	index := int(stdmath.Round(intensity*float64(rampLen))) - 1

	// Clamp to valid range
	if index < 0 {
		index = 0
	}
	if index >= rampLen {
		index = rampLen - 1
	}

	r := rune(s.LuminanceRamp[index])

	// Return color if ColorRamp is defined and has matching index
	var color canvas.Color
	if len(s.ColorRamp) > index {
		color = s.ColorRamp[index]
	}

	return r, color
}

// SetColorRamp sets the color gradient for shading (same length as LuminanceRamp).
func (s *Shader) SetColorRamp(colors []canvas.Color) {
	s.ColorRamp = colors
}

// IntensityToGray maps an intensity value to a grayscale color value (0-255).
func IntensityToGray(intensity float64) uint8 {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}
	return uint8(intensity * 255)
}
