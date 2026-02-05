package math

import "math"

// Mat4 represents a 4x4 matrix in row-major order.
// Index mapping: m[row*4 + col]
//
//	[0]  [1]  [2]  [3]
//	[4]  [5]  [6]  [7]
//	[8]  [9]  [10] [11]
//	[12] [13] [14] [15]
type Mat4 [16]float64

// Identity returns a 4x4 identity matrix.
func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Zero returns a zero matrix.
func Zero() Mat4 {
	return Mat4{}
}

// Perspective creates a perspective projection matrix.
// fovDegrees is the vertical field of view in degrees.
// aspect is the aspect ratio (width / height).
// near and far are the clipping planes.
// Uses standard OpenGL-style matrix (NDC Z in [-1, 1]).
func Perspective(fovDegrees, aspect, near, far float64) Mat4 {
	fovRad := (fovDegrees / 2.0) * (math.Pi / 180.0)
	tanHalfFov := math.Tan(fovRad)

	var m Mat4
	m[0] = (1.0 / tanHalfFov) / aspect // m[0][0]
	m[5] = 1.0 / tanHalfFov            // m[1][1]
	m[10] = -(far + near) / (far - near)
	m[11] = -2.0 * far * near / (far - near)
	m[14] = -1.0

	return m
}

// Orthographic creates an orthographic projection matrix.
func Orthographic(left, right, bottom, top, near, far float64) Mat4 {
	var m Mat4
	m[0] = 2.0 / (right - left)
	m[5] = 2.0 / (top - bottom)
	m[10] = -2.0 / (far - near)
	m[3] = -(right + left) / (right - left)
	m[7] = -(top + bottom) / (top - bottom)
	m[11] = -(far + near) / (far - near)
	m[15] = 1.0
	return m
}

// LookAt creates a view matrix looking from eye to target.
func LookAt(eye, target, up Vec3) Mat4 {
	// Forward vector (from target to eye, camera looks down -Z)
	f := target.Sub(eye).Normalize()
	// Right vector
	r := f.Cross(up).Normalize()
	// Recalculate up vector
	u := r.Cross(f)

	return Mat4{
		r.X, r.Y, r.Z, -r.Dot(eye),
		u.X, u.Y, u.Z, -u.Dot(eye),
		-f.X, -f.Y, -f.Z, f.Dot(eye),
		0, 0, 0, 1,
	}
}

// RotateX creates a rotation matrix around the X axis.
// angle is in degrees.
//
// Ported from ascii-graphics-3d/src/agm.cpp:31-43
func RotateX(degrees float64) Mat4 {
	radians := degrees * (math.Pi / 180.0)
	c := math.Cos(radians)
	s := math.Sin(radians)

	return Mat4{
		1, 0, 0, 0,
		0, c, -s, 0,
		0, s, c, 0,
		0, 0, 0, 1,
	}
}

// RotateY creates a rotation matrix around the Y axis.
// angle is in degrees.
//
// Ported from ascii-graphics-3d/src/agm.cpp:48-60
func RotateY(degrees float64) Mat4 {
	radians := degrees * (math.Pi / 180.0)
	c := math.Cos(radians)
	s := math.Sin(radians)

	return Mat4{
		c, 0, s, 0,
		0, 1, 0, 0,
		-s, 0, c, 0,
		0, 0, 0, 1,
	}
}

// RotateZ creates a rotation matrix around the Z axis.
// angle is in degrees.
//
// Ported from ascii-graphics-3d/src/agm.cpp:65-77
func RotateZ(degrees float64) Mat4 {
	radians := degrees * (math.Pi / 180.0)
	c := math.Cos(radians)
	s := math.Sin(radians)

	return Mat4{
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Translate creates a translation matrix.
func Translate(x, y, z float64) Mat4 {
	return Mat4{
		1, 0, 0, x,
		0, 1, 0, y,
		0, 0, 1, z,
		0, 0, 0, 1,
	}
}

// Scale creates a uniform scaling matrix.
func Scale(s float64) Mat4 {
	return Mat4{
		s, 0, 0, 0,
		0, s, 0, 0,
		0, 0, s, 0,
		0, 0, 0, 1,
	}
}

// Scale3 creates a non-uniform scaling matrix.
func Scale3(x, y, z float64) Mat4 {
	return Mat4{
		x, 0, 0, 0,
		0, y, 0, 0,
		0, 0, z, 0,
		0, 0, 0, 1,
	}
}

// Mul multiplies two 4x4 matrices (this * other).
func (m Mat4) Mul(other Mat4) Mat4 {
	var result Mat4
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			sum := 0.0
			for k := 0; k < 4; k++ {
				sum += m[row*4+k] * other[k*4+col]
			}
			result[row*4+col] = sum
		}
	}
	return result
}

// MulVec4 multiplies a matrix by a 4D vector.
func (m Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		X: m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]*v.W,
		Y: m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]*v.W,
		Z: m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]*v.W,
		W: m[12]*v.X + m[13]*v.Y + m[14]*v.Z + m[15]*v.W,
	}
}

// MulVec3 multiplies a matrix by a 3D vector (w=1), performs perspective divide.
// This is the equivalent of mult4() from the reference implementation.
//
// Ported from ascii-graphics-3d/src/agm.cpp:262-287
func (m Mat4) MulVec3(v Vec3) Vec3 {
	x := m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]
	y := m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]
	z := m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]
	w := m[12]*v.X + m[13]*v.Y + m[14]*v.Z + m[15]

	if w != 0 {
		return Vec3{X: x / w, Y: y / w, Z: z / w}
	}
	return Vec3{X: x, Y: y, Z: z}
}

// MulVec3Dir multiplies a matrix by a 3D direction vector (w=0, no translation).
// Use this for transforming normals and directions.
func (m Mat4) MulVec3Dir(v Vec3) Vec3 {
	return Vec3{
		X: m[0]*v.X + m[1]*v.Y + m[2]*v.Z,
		Y: m[4]*v.X + m[5]*v.Y + m[6]*v.Z,
		Z: m[8]*v.X + m[9]*v.Y + m[10]*v.Z,
	}
}

// Get returns the element at row, col.
func (m Mat4) Get(row, col int) float64 {
	return m[row*4+col]
}

// Set sets the element at row, col.
func (m *Mat4) Set(row, col int, val float64) {
	m[row*4+col] = val
}
