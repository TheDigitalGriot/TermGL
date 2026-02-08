package render

import (
	"math"

	"github.com/fogleman/fauxgl"
)

// Light represents directional, point, or ambient lighting.
// Architecture doc Section 2.4
type Light struct {
	Direction fauxgl.Vector
	Color     fauxgl.Color
	Intensity float64
}

// NewDirectionalLight creates a directional light.
func NewDirectionalLight(dirX, dirY, dirZ float64, color fauxgl.Color, intensity float64) Light {
	dir := fauxgl.V(dirX, dirY, dirZ).Normalize()
	return Light{
		Direction: dir,
		Color:     color,
		Intensity: intensity,
	}
}

// FlatShader implements fauxgl.Shader for flat shading with lighting.
type FlatShader struct {
	Matrix fauxgl.Matrix
	Lights []Light
}

// NewFlatShader creates a new flat shader.
func NewFlatShader(matrix fauxgl.Matrix, lights []Light) *FlatShader {
	return &FlatShader{
		Matrix: matrix,
		Lights: lights,
	}
}

// Vertex implements the vertex shader stage.
func (s *FlatShader) Vertex(vertex fauxgl.Vertex) fauxgl.Vertex {
	vertex.Output = s.Matrix.MulPositionW(vertex.Position)
	return vertex
}

// Fragment implements the fragment shader stage.
func (s *FlatShader) Fragment(v fauxgl.Vertex) fauxgl.Color {
	// Compute lighting
	color := fauxgl.Color{R: 0, G: 0, B: 0, A: 1}

	// Normalize the normal vector
	normal := v.Normal.Normalize()

	// Accumulate light contribution from each light source
	for _, light := range s.Lights {
		// Compute diffuse term (Lambertian reflection)
		// dot = max(0, normal · lightDir)
		dot := normal.Dot(light.Direction)
		if dot < 0 {
			dot = 0
		}

		// Add light contribution
		color.R += light.Color.R * light.Intensity * dot
		color.G += light.Color.G * light.Intensity * dot
		color.B += light.Color.B * light.Intensity * dot
	}

	// Clamp to [0, 1]
	color.R = math.Min(1, color.R)
	color.G = math.Min(1, color.G)
	color.B = math.Min(1, color.B)

	return color
}
