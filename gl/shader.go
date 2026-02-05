package gl

import (
	stdmath "math"

	"github.com/charmbracelet/termgl/math"
)

// DefaultLuminanceRamp is the default character gradient for shading.
// Characters go from darkest (first) to brightest (last).
// Ported from ascii-graphics-3d: ".:+*=#%@"
const DefaultLuminanceRamp = " .:-=+*#%@"

// Shader calculates the visual appearance of surfaces.
type Shader struct {
	LuminanceRamp string

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
	// Calculate diffuse intensity using dot product
	diffuse := stdmath.Abs(normal.Dot(s.DirectionalLight.Direction))
	diffuse *= s.DirectionalLight.Intensity

	// Add ambient
	intensity := diffuse
	if s.AmbientLight != nil {
		intensity = intensity*(1-s.AmbientLight.Intensity) + s.AmbientLight.Intensity
	}

	return s.intensityToRune(intensity)
}

// CalculateSmoothShade computes the shade for smooth (Gouraud) shading.
// It interpolates vertex normals using barycentric coordinates.
//
// Ported from ascii-graphics-3d/src/Screen.cpp:72-96
func (s *Shader) CalculateSmoothShade(n1, n2, n3 math.Vec3, w1, w2, w3 float64) rune {
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

	return s.intensityToRune(intensity)
}

// intensityToRune maps an intensity value (0.0-1.0) to a character.
func (s *Shader) intensityToRune(intensity float64) rune {
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

	return rune(s.LuminanceRamp[index])
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
