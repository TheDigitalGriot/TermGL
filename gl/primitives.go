package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// NewCube creates a unit cube centered at the origin.
// Returns a mesh with 12 triangles (2 per face).
//
// Ported from ascii-graphics-3d/src/Mesh.cpp:248-313
func NewCube() *Mesh {
	m := NewMesh()

	// Define the 8 vertices of a unit cube (-1 to 1)
	v := []math.Vec3{
		{X: -1, Y: -1, Z: -1}, // 0: left-bottom-back
		{X: 1, Y: -1, Z: -1},  // 1: right-bottom-back
		{X: 1, Y: 1, Z: -1},   // 2: right-top-back
		{X: -1, Y: 1, Z: -1},  // 3: left-top-back
		{X: -1, Y: -1, Z: 1},  // 4: left-bottom-front
		{X: 1, Y: -1, Z: 1},   // 5: right-bottom-front
		{X: 1, Y: 1, Z: 1},    // 6: right-top-front
		{X: -1, Y: 1, Z: 1},   // 7: left-top-front
	}

	// Define face normals
	front := math.Vec3{X: 0, Y: 0, Z: 1}
	back := math.Vec3{X: 0, Y: 0, Z: -1}
	left := math.Vec3{X: -1, Y: 0, Z: 0}
	right := math.Vec3{X: 1, Y: 0, Z: 0}
	top := math.Vec3{X: 0, Y: 1, Z: 0}
	bottom := math.Vec3{X: 0, Y: -1, Z: 0}

	// Helper to create a triangle
	makeTri := func(p0, p1, p2 math.Vec3, n math.Vec3) Triangle {
		return Triangle{
			Vertices: [3]Vertex{
				{Position: p0, Normal: n},
				{Position: p1, Normal: n},
				{Position: p2, Normal: n},
			},
			FaceNormal: n,
		}
	}

	// Front face (z = 1)
	m.AddTriangle(makeTri(v[4], v[5], v[6], front))
	m.AddTriangle(makeTri(v[4], v[6], v[7], front))

	// Back face (z = -1)
	m.AddTriangle(makeTri(v[1], v[0], v[3], back))
	m.AddTriangle(makeTri(v[1], v[3], v[2], back))

	// Left face (x = -1)
	m.AddTriangle(makeTri(v[0], v[4], v[7], left))
	m.AddTriangle(makeTri(v[0], v[7], v[3], left))

	// Right face (x = 1)
	m.AddTriangle(makeTri(v[5], v[1], v[2], right))
	m.AddTriangle(makeTri(v[5], v[2], v[6], right))

	// Top face (y = 1)
	m.AddTriangle(makeTri(v[7], v[6], v[2], top))
	m.AddTriangle(makeTri(v[7], v[2], v[3], top))

	// Bottom face (y = -1)
	m.AddTriangle(makeTri(v[0], v[1], v[5], bottom))
	m.AddTriangle(makeTri(v[0], v[5], v[4], bottom))

	return m
}

// NewPlane creates a plane in the XZ plane centered at origin.
func NewPlane(width, depth float64) *Mesh {
	m := NewMesh()

	hw := width / 2
	hd := depth / 2

	v := []math.Vec3{
		{X: -hw, Y: 0, Z: -hd},
		{X: hw, Y: 0, Z: -hd},
		{X: hw, Y: 0, Z: hd},
		{X: -hw, Y: 0, Z: hd},
	}

	up := math.Vec3{X: 0, Y: 1, Z: 0}

	m.AddTriangle(Triangle{
		Vertices: [3]Vertex{
			{Position: v[0], Normal: up},
			{Position: v[1], Normal: up},
			{Position: v[2], Normal: up},
		},
		FaceNormal: up,
	})
	m.AddTriangle(Triangle{
		Vertices: [3]Vertex{
			{Position: v[0], Normal: up},
			{Position: v[2], Normal: up},
			{Position: v[3], Normal: up},
		},
		FaceNormal: up,
	})

	return m
}

// NewPyramid creates a 4-sided pyramid.
func NewPyramid() *Mesh {
	m := NewMesh()

	// Base vertices (square at y = -1)
	base := []math.Vec3{
		{X: -1, Y: -1, Z: -1},
		{X: 1, Y: -1, Z: -1},
		{X: 1, Y: -1, Z: 1},
		{X: -1, Y: -1, Z: 1},
	}
	apex := math.Vec3{X: 0, Y: 1, Z: 0}

	// Helper to compute triangle normal
	triNormal := func(p0, p1, p2 math.Vec3) math.Vec3 {
		e1 := p1.Sub(p0)
		e2 := p2.Sub(p0)
		return e1.Cross(e2).Normalize()
	}

	// Four side faces
	for i := 0; i < 4; i++ {
		p0 := base[i]
		p1 := base[(i+1)%4]
		n := triNormal(p0, p1, apex)
		m.AddTriangle(Triangle{
			Vertices: [3]Vertex{
				{Position: p0, Normal: n},
				{Position: p1, Normal: n},
				{Position: apex, Normal: n},
			},
			FaceNormal: n,
		})
	}

	// Bottom face (two triangles)
	bottomN := math.Vec3{X: 0, Y: -1, Z: 0}
	m.AddTriangle(Triangle{
		Vertices: [3]Vertex{
			{Position: base[0], Normal: bottomN},
			{Position: base[2], Normal: bottomN},
			{Position: base[1], Normal: bottomN},
		},
		FaceNormal: bottomN,
	})
	m.AddTriangle(Triangle{
		Vertices: [3]Vertex{
			{Position: base[0], Normal: bottomN},
			{Position: base[3], Normal: bottomN},
			{Position: base[2], Normal: bottomN},
		},
		FaceNormal: bottomN,
	})

	return m
}
