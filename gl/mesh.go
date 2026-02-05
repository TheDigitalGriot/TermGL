// Package gl provides 3D rendering pipeline for terminal graphics.
package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// Vertex represents a vertex with position, normal, and UV coordinates.
type Vertex struct {
	Position math.Vec3
	Normal   math.Vec3
	UV       math.Vec2
}

// Triangle represents a triangle with three vertices and a face normal.
type Triangle struct {
	Vertices   [3]Vertex
	FaceNormal math.Vec3
}

// ComputeFaceNormal calculates the face normal from the triangle vertices.
func (t *Triangle) ComputeFaceNormal() {
	v0 := t.Vertices[0].Position
	v1 := t.Vertices[1].Position
	v2 := t.Vertices[2].Position

	// Edge vectors
	e1 := v1.Sub(v0)
	e2 := v2.Sub(v0)

	// Cross product gives normal direction
	t.FaceNormal = e1.Cross(e2).Normalize()
}

// Mesh represents a 3D mesh as a collection of triangles.
type Mesh struct {
	Triangles []Triangle
	Transform math.Transform

	// Cached transformation amounts for local rotation (like reference impl)
	transAmt math.Vec3
	rotAmt   math.Vec3
}

// NewMesh creates an empty mesh.
func NewMesh() *Mesh {
	return &Mesh{
		Triangles: make([]Triangle, 0),
		Transform: math.NewTransform(),
	}
}

// AddTriangle adds a triangle to the mesh.
func (m *Mesh) AddTriangle(t Triangle) {
	m.Triangles = append(m.Triangles, t)
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int {
	return len(m.Triangles)
}

// SetPosition sets the mesh position.
func (m *Mesh) SetPosition(x, y, z float64) {
	m.Transform.Position = math.Vec3{X: x, Y: y, Z: z}
}

// SetRotation sets the mesh rotation in degrees.
func (m *Mesh) SetRotation(x, y, z float64) {
	m.Transform.Rotation = math.Vec3{X: x, Y: y, Z: z}
}

// SetScale sets the mesh uniform scale.
func (m *Mesh) SetScale(s float64) {
	m.Transform.Scale = math.Vec3{X: s, Y: s, Z: s}
}

// ComputeFaceNormals calculates face normals for all triangles.
func (m *Mesh) ComputeFaceNormals() {
	for i := range m.Triangles {
		m.Triangles[i].ComputeFaceNormal()
	}
}

// ComputeVertexNormals calculates smooth vertex normals by averaging face normals.
// This should be called after ComputeFaceNormals if smooth shading is desired.
func (m *Mesh) ComputeVertexNormals() {
	// Build a map of position -> accumulated normal
	type posKey struct {
		x, y, z float64
	}
	normalAccum := make(map[posKey]math.Vec3)
	normalCount := make(map[posKey]int)

	// Accumulate face normals for each vertex position
	for _, tri := range m.Triangles {
		for _, vert := range tri.Vertices {
			key := posKey{vert.Position.X, vert.Position.Y, vert.Position.Z}
			normalAccum[key] = normalAccum[key].Add(tri.FaceNormal)
			normalCount[key]++
		}
	}

	// Average and normalize
	for key := range normalAccum {
		count := float64(normalCount[key])
		normalAccum[key] = normalAccum[key].Mul(1.0 / count).Normalize()
	}

	// Apply averaged normals back to vertices
	for i := range m.Triangles {
		for j := range m.Triangles[i].Vertices {
			pos := m.Triangles[i].Vertices[j].Position
			key := posKey{pos.X, pos.Y, pos.Z}
			m.Triangles[i].Vertices[j].Normal = normalAccum[key]
		}
	}
}

// Clone creates a deep copy of the mesh.
func (m *Mesh) Clone() *Mesh {
	clone := &Mesh{
		Triangles: make([]Triangle, len(m.Triangles)),
		Transform: m.Transform,
		transAmt:  m.transAmt,
		rotAmt:    m.rotAmt,
	}
	copy(clone.Triangles, m.Triangles)
	return clone
}
