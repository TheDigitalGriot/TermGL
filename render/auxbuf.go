package render

import (
	"math"

	"github.com/fogleman/fauxgl"
)

// AuxBuffers holds auxiliary render outputs for geometry-aware encoding.
// Architecture doc Section 2.5
type AuxBuffers struct {
	Width     int
	Height    int
	DepthMap  []float64       // linearized depth per pixel
	NormalMap []fauxgl.Vector // world-space normal per pixel
}

// AuxBufferShader captures depth and normals during rasterization.
// Architecture doc Section 2.5
type AuxBufferShader struct {
	Matrix    fauxgl.Matrix
	Lights    []Light
	Width     int
	DepthMap  []float64
	NormalMap []fauxgl.Vector
}

// NewAuxBufferShader creates a new auxiliary buffer shader.
func NewAuxBufferShader(matrix fauxgl.Matrix, lights []Light, width int, depthMap []float64, normalMap []fauxgl.Vector) *AuxBufferShader {
	return &AuxBufferShader{
		Matrix:    matrix,
		Lights:    lights,
		Width:     width,
		DepthMap:  depthMap,
		NormalMap: normalMap,
	}
}

// Vertex implements the vertex shader stage.
func (s *AuxBufferShader) Vertex(vertex fauxgl.Vertex) fauxgl.Vertex {
	vertex.Output = s.Matrix.MulPositionW(vertex.Position)
	return vertex
}

// Fragment implements the fragment shader stage.
// It computes lighting AND stores depth and normal data.
func (s *AuxBufferShader) Fragment(v fauxgl.Vertex) fauxgl.Color {
	// Store auxiliary data at this fragment's screen position
	x, y := int(v.Output.X), int(v.Output.Y)

	// Bounds check to prevent out-of-range writes
	if x >= 0 && y >= 0 && x < s.Width && y < len(s.DepthMap)/s.Width {
		idx := y*s.Width + x
		if idx < len(s.DepthMap) {
			s.DepthMap[idx] = v.Output.Z
		}
		if idx < len(s.NormalMap) {
			s.NormalMap[idx] = v.Normal
		}
	}

	// Compute lighting (same as FlatShader)
	color := fauxgl.Color{R: 0, G: 0, B: 0, A: 1}
	normal := v.Normal.Normalize()

	for _, light := range s.Lights {
		dot := normal.Dot(light.Direction)
		if dot < 0 {
			dot = 0
		}

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
