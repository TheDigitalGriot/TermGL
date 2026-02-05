// Package math provides linear algebra types for 3D graphics.
package math

import "math"

// Vec2 represents a 2D vector.
type Vec2 struct {
	X, Y float64
}

// NewVec2 creates a new 2D vector.
func NewVec2(x, y float64) Vec2 {
	return Vec2{X: x, Y: y}
}

// Add returns the sum of two vectors.
func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Sub returns the difference of two vectors.
func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

// Mul returns the vector scaled by a scalar.
func (v Vec2) Mul(s float64) Vec2 {
	return Vec2{X: v.X * s, Y: v.Y * s}
}

// Vec3 represents a 3D vector.
type Vec3 struct {
	X, Y, Z float64
}

// NewVec3 creates a new 3D vector.
func NewVec3(x, y, z float64) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

// Add returns the sum of two vectors.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Sub returns the difference of two vectors.
func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Mul returns the vector scaled by a scalar.
func (v Vec3) Mul(s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// Dot returns the dot product of two vectors.
func (v Vec3) Dot(other Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross returns the cross product of two vectors.
func (v Vec3) Cross(other Vec3) Vec3 {
	return Vec3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Length returns the magnitude of the vector.
func (v Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// LengthSquared returns the squared magnitude (avoids sqrt).
func (v Vec3) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

// Normalize returns a unit vector in the same direction.
func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return Vec3{X: v.X / l, Y: v.Y / l, Z: v.Z / l}
}

// Lerp returns linear interpolation between two vectors.
func (v Vec3) Lerp(other Vec3, t float64) Vec3 {
	return Vec3{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
		Z: v.Z + (other.Z-v.Z)*t,
	}
}

// Negate returns the negated vector.
func (v Vec3) Negate() Vec3 {
	return Vec3{X: -v.X, Y: -v.Y, Z: -v.Z}
}

// Vec4 represents a 4D vector (homogeneous coordinates).
type Vec4 struct {
	X, Y, Z, W float64
}

// NewVec4 creates a new 4D vector.
func NewVec4(x, y, z, w float64) Vec4 {
	return Vec4{X: x, Y: y, Z: z, W: w}
}

// Vec3ToVec4 converts a Vec3 to Vec4 with w=1 (point).
func Vec3ToVec4(v Vec3) Vec4 {
	return Vec4{X: v.X, Y: v.Y, Z: v.Z, W: 1.0}
}

// ToVec3 converts Vec4 to Vec3 by performing perspective divide.
func (v Vec4) ToVec3() Vec3 {
	if v.W != 0 {
		return Vec3{X: v.X / v.W, Y: v.Y / v.W, Z: v.Z / v.W}
	}
	return Vec3{X: v.X, Y: v.Y, Z: v.Z}
}

// Add returns the sum of two vectors.
func (v Vec4) Add(other Vec4) Vec4 {
	return Vec4{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z, W: v.W + other.W}
}

// Mul returns the vector scaled by a scalar.
func (v Vec4) Mul(s float64) Vec4 {
	return Vec4{X: v.X * s, Y: v.Y * s, Z: v.Z * s, W: v.W * s}
}
