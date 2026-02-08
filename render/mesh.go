package render

import (
	"github.com/fogleman/fauxgl"
)

// LoadOBJ loads an OBJ file via FauxGL's loader.
// Supports normals, UVs, and arbitrary polygon triangulation.
func LoadOBJ(path string) (*fauxgl.Mesh, error) {
	mesh, err := fauxgl.LoadOBJ(path)
	if err != nil {
		return nil, err
	}
	return mesh, nil
}

// LoadSTL loads an STL file via FauxGL's loader.
func LoadSTL(path string) (*fauxgl.Mesh, error) {
	mesh, err := fauxgl.LoadSTL(path)
	if err != nil {
		return nil, err
	}
	return mesh, nil
}

// NewCubeMesh creates a simple cube mesh for testing.
func NewCubeMesh() *fauxgl.Mesh {
	return fauxgl.NewCubeForBox(fauxgl.Box{
		Min: fauxgl.V(-1, -1, -1),
		Max: fauxgl.V(1, 1, 1),
	})
}

// NewSphereMesh creates a sphere mesh for testing.
func NewSphereMesh(radius float64, detail int) *fauxgl.Mesh {
	return fauxgl.NewSphere(detail)
}
