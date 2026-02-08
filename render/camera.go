package render

import (
	"math"

	"github.com/fogleman/fauxgl"
)

// Camera provides perspective projection with configurable FOV, near/far.
// Architecture doc Section 2.4
type Camera struct {
	Position fauxgl.Vector
	Target   fauxgl.Vector
	Up       fauxgl.Vector
	FOV      float64 // degrees
	Near     float64
	Far      float64
}

// NewCamera creates a camera with default settings.
func NewCamera() Camera {
	return Camera{
		Position: fauxgl.V(0, 0, 3),
		Target:   fauxgl.V(0, 0, 0),
		Up:       fauxgl.V(0, 1, 0),
		FOV:      50.0,
		Near:     0.1,
		Far:      100.0,
	}
}

// ViewMatrix computes the view matrix (camera transform).
func (c *Camera) ViewMatrix() fauxgl.Matrix {
	return fauxgl.LookAt(c.Position, c.Target, c.Up)
}

// ProjectionMatrix computes the projection matrix.
func (c *Camera) ProjectionMatrix(aspect float64) fauxgl.Matrix {
	return fauxgl.Perspective(c.FOV, aspect, c.Near, c.Far)
}

// SetPosition sets the camera position.
func (c *Camera) SetPosition(x, y, z float64) {
	c.Position = fauxgl.V(x, y, z)
}

// SetTarget sets the camera target (look-at point).
func (c *Camera) SetTarget(x, y, z float64) {
	c.Target = fauxgl.V(x, y, z)
}

// LookAt sets both position and target for the camera.
func (c *Camera) LookAt(eyeX, eyeY, eyeZ, targetX, targetY, targetZ float64) {
	c.Position = fauxgl.V(eyeX, eyeY, eyeZ)
	c.Target = fauxgl.V(targetX, targetY, targetZ)
}

// OrbitAround rotates the camera around the target point.
// angle is in radians, height is the vertical offset.
func (c *Camera) OrbitAround(angle, distance, height float64) {
	c.Position = fauxgl.V(
		c.Target.X+distance*math.Cos(angle),
		c.Target.Y+height,
		c.Target.Z+distance*math.Sin(angle),
	)
}
