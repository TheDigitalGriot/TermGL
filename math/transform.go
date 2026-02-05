package math

// Transform represents a 3D transform with position, rotation, and scale.
type Transform struct {
	Position Vec3
	Rotation Vec3 // Euler angles in degrees
	Scale    Vec3
}

// NewTransform creates a transform at the origin with unit scale.
func NewTransform() Transform {
	return Transform{
		Position: Vec3{},
		Rotation: Vec3{},
		Scale:    Vec3{X: 1, Y: 1, Z: 1},
	}
}

// Matrix returns the combined transformation matrix (Scale * Rotation * Translation).
func (t Transform) Matrix() Mat4 {
	// Build rotation matrix (Y * X * Z order, matching reference implementation)
	rotY := RotateY(t.Rotation.Y)
	rotX := RotateX(t.Rotation.X)
	rotZ := RotateZ(t.Rotation.Z)
	rot := rotY.Mul(rotX).Mul(rotZ)

	// Build scale matrix
	scale := Scale3(t.Scale.X, t.Scale.Y, t.Scale.Z)

	// Build translation matrix
	trans := Translate(t.Position.X, t.Position.Y, t.Position.Z)

	// Combine: Translation * Rotation * Scale
	return trans.Mul(rot).Mul(scale)
}

// SetRotationDegrees sets rotation from degrees.
func (t *Transform) SetRotationDegrees(x, y, z float64) {
	t.Rotation = Vec3{X: x, Y: y, Z: z}
}

// SetPosition sets the position.
func (t *Transform) SetPosition(x, y, z float64) {
	t.Position = Vec3{X: x, Y: y, Z: z}
}

// SetUniformScale sets uniform scale.
func (t *Transform) SetUniformScale(s float64) {
	t.Scale = Vec3{X: s, Y: s, Z: s}
}

// CalcZ calculates the Z value from x,y using the plane equation.
// c is the cross product (normal) of the plane, v is a point on the plane.
//
// Ported from ascii-graphics-3d/src/agm.cpp:326-330
func CalcZ(x, y float64, c, v Vec3) float64 {
	k := c.Dot(v)
	if c.Z == 0 {
		return 0
	}
	return (c.X*x + c.Y*y - k) / (-c.Z)
}

// CalcBary calculates barycentric coordinates for point (x,y) in triangle (p1,p2,p3).
// Returns w1, w2, w3 such that P = w1*p1 + w2*p2 + w3*p3.
//
// Ported from ascii-graphics-3d/src/agm.cpp:337-344
func CalcBary(p1, p2, p3 Vec3, x, y int) (w1, w2, w3 float64) {
	fx, fy := float64(x), float64(y)

	denom := (p2.Y-p3.Y)*(p1.X-p3.X) + (p3.X-p2.X)*(p1.Y-p3.Y)
	if denom == 0 {
		return 0, 0, 0
	}

	w1 = ((p2.Y-p3.Y)*(fx-p3.X) + (p3.X-p2.X)*(fy-p3.Y)) / denom
	w2 = ((p3.Y-p1.Y)*(fx-p3.X) + (p1.X-p3.X)*(fy-p3.Y)) / denom
	w3 = 1.0 - w1 - w2

	return w1, w2, w3
}
