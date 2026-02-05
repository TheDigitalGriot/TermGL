package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// NewCube creates a unit cube centered at the origin.
// Returns a mesh with 12 triangles (2 per face).
// Each face has proper UV coordinates for texture mapping.
//
// Ported from ascii-graphics-3d/src/Mesh.cpp:248-313
// Enhanced with UV coordinates matching TermGL-C-Plus.
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

	// Standard quad UVs (0-1 range)
	uv00 := math.Vec2{X: 0, Y: 0} // bottom-left
	uv10 := math.Vec2{X: 1, Y: 0} // bottom-right
	uv01 := math.Vec2{X: 0, Y: 1} // top-left
	uv11 := math.Vec2{X: 1, Y: 1} // top-right

	// Helper to create a triangle with UVs
	makeTri := func(p0, p1, p2 math.Vec3, n math.Vec3, uv0, uv1, uv2 math.Vec2) Triangle {
		return Triangle{
			Vertices: [3]Vertex{
				{Position: p0, Normal: n, UV: uv0},
				{Position: p1, Normal: n, UV: uv1},
				{Position: p2, Normal: n, UV: uv2},
			},
			FaceNormal: n,
		}
	}

	// Front face (z = 1) - looking at face from outside
	// v4=bottom-left, v5=bottom-right, v6=top-right, v7=top-left
	m.AddTriangle(makeTri(v[4], v[5], v[6], front, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[4], v[6], v[7], front, uv00, uv11, uv01))

	// Back face (z = -1) - looking at face from outside (reversed winding)
	// v1=bottom-left, v0=bottom-right, v3=top-right, v2=top-left
	m.AddTriangle(makeTri(v[1], v[0], v[3], back, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[1], v[3], v[2], back, uv00, uv11, uv01))

	// Left face (x = -1) - looking at face from outside
	// v0=bottom-left, v4=bottom-right, v7=top-right, v3=top-left
	m.AddTriangle(makeTri(v[0], v[4], v[7], left, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[0], v[7], v[3], left, uv00, uv11, uv01))

	// Right face (x = 1) - looking at face from outside
	// v5=bottom-left, v1=bottom-right, v2=top-right, v6=top-left
	m.AddTriangle(makeTri(v[5], v[1], v[2], right, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[5], v[2], v[6], right, uv00, uv11, uv01))

	// Top face (y = 1) - looking at face from above
	// v7=bottom-left, v6=bottom-right, v2=top-right, v3=top-left
	m.AddTriangle(makeTri(v[7], v[6], v[2], top, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[7], v[2], v[3], top, uv00, uv11, uv01))

	// Bottom face (y = -1) - looking at face from below
	// v0=bottom-left, v1=bottom-right, v5=top-right, v4=top-left
	m.AddTriangle(makeTri(v[0], v[1], v[5], bottom, uv00, uv10, uv11))
	m.AddTriangle(makeTri(v[0], v[5], v[4], bottom, uv00, uv11, uv01))

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
