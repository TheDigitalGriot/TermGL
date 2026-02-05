package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// ProjectionType determines perspective or orthographic projection.
type ProjectionType int

const (
	Perspective ProjectionType = iota
	Orthographic
)

// Camera represents a 3D camera with view and projection transforms.
type Camera struct {
	Position math.Vec3
	Target   math.Vec3
	Up       math.Vec3

	Projection ProjectionType
	FOV        float64 // Field of view in degrees (for perspective)
	Near       float64 // Near clipping plane
	Far        float64 // Far clipping plane
	Aspect     float64 // Aspect ratio (width/height)

	// Orthographic bounds
	OrthoWidth  float64
	OrthoHeight float64

	// Cached projection matrix
	projMat math.Mat4
}

// NewPerspectiveCamera creates a perspective camera.
// fov is in degrees, aspect is width/height.
func NewPerspectiveCamera(fov, aspect, near, far float64) *Camera {
	c := &Camera{
		Position:   math.Vec3{X: 0, Y: 0, Z: 0},
		Target:     math.Vec3{X: 0, Y: 0, Z: -1},
		Up:         math.Vec3{X: 0, Y: 1, Z: 0},
		Projection: Perspective,
		FOV:        fov,
		Near:       near,
		Far:        far,
		Aspect:     aspect,
	}
	c.updateProjectionMatrix()
	return c
}

// NewOrthographicCamera creates an orthographic camera.
func NewOrthographicCamera(width, height, near, far float64) *Camera {
	c := &Camera{
		Position:    math.Vec3{X: 0, Y: 0, Z: 0},
		Target:      math.Vec3{X: 0, Y: 0, Z: -1},
		Up:          math.Vec3{X: 0, Y: 1, Z: 0},
		Projection:  Orthographic,
		Near:        near,
		Far:         far,
		OrthoWidth:  width,
		OrthoHeight: height,
	}
	c.updateProjectionMatrix()
	return c
}

// updateProjectionMatrix recalculates the projection matrix.
func (c *Camera) updateProjectionMatrix() {
	if c.Projection == Perspective {
		c.projMat = math.Perspective(c.FOV, c.Aspect, c.Near, c.Far)
	} else {
		hw := c.OrthoWidth / 2
		hh := c.OrthoHeight / 2
		c.projMat = math.Orthographic(-hw, hw, -hh, hh, c.Near, c.Far)
	}
}

// ViewMatrix returns the view transformation matrix.
func (c *Camera) ViewMatrix() math.Mat4 {
	return math.LookAt(c.Position, c.Target, c.Up)
}

// ProjectionMatrix returns the projection matrix.
func (c *Camera) ProjectionMatrix() math.Mat4 {
	return c.projMat
}

// ViewProjectionMatrix returns the combined view-projection matrix.
func (c *Camera) ViewProjectionMatrix() math.Mat4 {
	return c.projMat.Mul(c.ViewMatrix())
}

// LookAt points the camera at a target position.
func (c *Camera) LookAt(target math.Vec3) {
	c.Target = target
}

// SetPosition sets the camera position.
func (c *Camera) SetPosition(x, y, z float64) {
	c.Position = math.Vec3{X: x, Y: y, Z: z}
}

// SetAspect updates the aspect ratio and recalculates projection.
func (c *Camera) SetAspect(aspect float64) {
	c.Aspect = aspect
	c.updateProjectionMatrix()
}

// SetFOV updates the field of view and recalculates projection.
func (c *Camera) SetFOV(fov float64) {
	c.FOV = fov
	c.updateProjectionMatrix()
}

// Forward returns the forward direction vector (from position toward target).
func (c *Camera) Forward() math.Vec3 {
	return c.Target.Sub(c.Position).Normalize()
}

// Right returns the right direction vector.
func (c *Camera) Right() math.Vec3 {
	return c.Forward().Cross(c.Up).Normalize()
}
