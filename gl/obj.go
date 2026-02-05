package gl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/termgl/math"
)

// LoadOBJFile loads a mesh from an OBJ file path.
func LoadOBJFile(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open OBJ file: %w", err)
	}
	defer f.Close()

	return LoadOBJ(f)
}

// LoadOBJ loads a mesh from an OBJ file reader.
// Supports:
//   - v (vertices)
//   - vn (vertex normals)
//   - vt (texture coordinates) - stored but not used
//   - f (faces) - triangles with formats: v, v/vt, v/vt/vn, v//vn
//
// Ported from ascii-graphics-3d/src/Mesh.cpp:71-234
func LoadOBJ(r io.Reader) (*Mesh, error) {
	mesh := NewMesh()

	var vertices []math.Vec3
	var normals []math.Vec3
	var texCoords []math.Vec2

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "v": // Vertex position
			if len(fields) < 4 {
				continue // Skip malformed vertex
			}
			x, _ := strconv.ParseFloat(fields[1], 64)
			y, _ := strconv.ParseFloat(fields[2], 64)
			z, _ := strconv.ParseFloat(fields[3], 64)
			vertices = append(vertices, math.Vec3{X: x, Y: y, Z: z})

		case "vn": // Vertex normal
			if len(fields) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 64)
			y, _ := strconv.ParseFloat(fields[2], 64)
			z, _ := strconv.ParseFloat(fields[3], 64)
			normals = append(normals, math.Vec3{X: x, Y: y, Z: z})

		case "vt": // Texture coordinate
			if len(fields) < 3 {
				continue
			}
			u, _ := strconv.ParseFloat(fields[1], 64)
			v, _ := strconv.ParseFloat(fields[2], 64)
			texCoords = append(texCoords, math.Vec2{X: u, Y: v})

		case "f": // Face
			if len(fields) < 4 {
				continue // Need at least 3 vertices for a triangle
			}

			// Parse face vertices
			var faceVerts []Vertex
			for i := 1; i < len(fields); i++ {
				v := parseFaceVertex(fields[i], vertices, normals, texCoords)
				faceVerts = append(faceVerts, v)
			}

			// Triangulate (fan triangulation for convex polygons)
			for i := 1; i < len(faceVerts)-1; i++ {
				tri := Triangle{
					Vertices: [3]Vertex{faceVerts[0], faceVerts[i], faceVerts[i+1]},
				}
				// Compute face normal from vertices
				tri.ComputeFaceNormal()
				mesh.AddTriangle(tri)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading OBJ: %w", err)
	}

	// If no vertex normals were provided, compute smooth normals
	hasNormals := len(normals) > 0
	if !hasNormals {
		mesh.ComputeVertexNormals()
	}

	return mesh, nil
}

// parseFaceVertex parses a face vertex specification.
// Formats: v, v/vt, v/vt/vn, v//vn
func parseFaceVertex(spec string, verts []math.Vec3, norms []math.Vec3, texs []math.Vec2) Vertex {
	parts := strings.Split(spec, "/")

	var v Vertex

	// Vertex index (1-based in OBJ)
	if len(parts) > 0 && parts[0] != "" {
		if idx, err := strconv.Atoi(parts[0]); err == nil {
			// Handle negative indices (relative to end)
			if idx < 0 {
				idx = len(verts) + idx + 1
			}
			if idx > 0 && idx <= len(verts) {
				v.Position = verts[idx-1]
			}
		}
	}

	// Texture coordinate index
	if len(parts) > 1 && parts[1] != "" {
		if idx, err := strconv.Atoi(parts[1]); err == nil {
			if idx < 0 {
				idx = len(texs) + idx + 1
			}
			if idx > 0 && idx <= len(texs) {
				v.UV = texs[idx-1]
			}
		}
	}

	// Normal index
	if len(parts) > 2 && parts[2] != "" {
		if idx, err := strconv.Atoi(parts[2]); err == nil {
			if idx < 0 {
				idx = len(norms) + idx + 1
			}
			if idx > 0 && idx <= len(norms) {
				v.Normal = norms[idx-1]
			}
		}
	}

	return v
}

// OBJOptions configures OBJ loading behavior.
type OBJOptions struct {
	FlipNormals bool    // Flip normal direction
	Scale       float64 // Scale factor to apply (0 = no scaling)
}

// LoadOBJWithOptions loads with custom options.
func LoadOBJWithOptions(r io.Reader, opts OBJOptions) (*Mesh, error) {
	mesh, err := LoadOBJ(r)
	if err != nil {
		return nil, err
	}

	// Apply scale
	if opts.Scale != 0 && opts.Scale != 1 {
		for i := range mesh.Triangles {
			for j := range mesh.Triangles[i].Vertices {
				mesh.Triangles[i].Vertices[j].Position =
					mesh.Triangles[i].Vertices[j].Position.Mul(opts.Scale)
			}
		}
	}

	// Flip normals
	if opts.FlipNormals {
		for i := range mesh.Triangles {
			mesh.Triangles[i].FaceNormal = mesh.Triangles[i].FaceNormal.Negate()
			for j := range mesh.Triangles[i].Vertices {
				mesh.Triangles[i].Vertices[j].Normal =
					mesh.Triangles[i].Vertices[j].Normal.Negate()
			}
		}
	}

	return mesh, nil
}
