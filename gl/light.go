package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// DirectionalLight represents an infinite-distance parallel light (like sunlight).
type DirectionalLight struct {
	Direction math.Vec3 // Direction the light is pointing (normalized)
	Intensity float64   // Light intensity (0.0 to 1.0)
}

// NewDirectionalLight creates a directional light.
// direction should point in the direction the light travels (toward objects).
func NewDirectionalLight(direction math.Vec3, intensity float64) *DirectionalLight {
	return &DirectionalLight{
		Direction: direction.Normalize(),
		Intensity: intensity,
	}
}

// DefaultDirectionalLight creates a light pointing into the screen (-Z).
// This matches the reference implementation's default.
func DefaultDirectionalLight() *DirectionalLight {
	return NewDirectionalLight(math.Vec3{X: 0, Y: 0, Z: -1}, 1.0)
}

// AmbientLight represents uniform ambient illumination.
type AmbientLight struct {
	Intensity float64 // Ambient light level (0.0 to 1.0)
}

// NewAmbientLight creates an ambient light.
func NewAmbientLight(intensity float64) *AmbientLight {
	return &AmbientLight{
		Intensity: intensity,
	}
}
