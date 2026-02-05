package gl

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/termgl/math"
)

// STL file format constants
const (
	stlHeaderSize    = 80
	stlTriangleSize  = 50 // 4*3 (normal) + 4*3*3 (vertices) + 2 (attribute)
	stlBinaryMinSize = stlHeaderSize + 4 // header + triangle count
)

// LoadSTLFile loads a mesh from an STL file (binary or ASCII format).
func LoadSTLFile(path string) (*Mesh, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return LoadSTL(file)
}

// LoadSTL loads a mesh from an STL reader (binary or ASCII format).
func LoadSTL(r io.Reader) (*Mesh, error) {
	// Read enough to detect format
	buf := bufio.NewReader(r)

	// Peek at the beginning to detect format
	header, err := buf.Peek(stlHeaderSize + 4)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Check if ASCII format (starts with "solid")
	if len(header) >= 5 && strings.HasPrefix(strings.ToLower(string(header[:5])), "solid") {
		// Could be ASCII, but need to verify
		// Some binary files also start with "solid"
		// Check if there's "facet" after potential name
		if isASCIISTL(header) {
			return loadSTLASCII(buf)
		}
	}

	// Default to binary format
	return loadSTLBinary(buf)
}

// LoadSTLWithOptions loads an STL file with custom options.
func LoadSTLWithOptions(r io.Reader, opts OBJOptions) (*Mesh, error) {
	mesh, err := LoadSTL(r)
	if err != nil {
		return nil, err
	}

	// Apply scale
	if opts.Scale != 0 && opts.Scale != 1 {
		for i := range mesh.Triangles {
			for j := range mesh.Triangles[i].Vertices {
				mesh.Triangles[i].Vertices[j].Position = mesh.Triangles[i].Vertices[j].Position.Mul(opts.Scale)
			}
		}
	}

	// Flip normals if requested
	if opts.FlipNormals {
		for i := range mesh.Triangles {
			mesh.Triangles[i].FaceNormal = mesh.Triangles[i].FaceNormal.Mul(-1)
			for j := range mesh.Triangles[i].Vertices {
				mesh.Triangles[i].Vertices[j].Normal = mesh.Triangles[i].Vertices[j].Normal.Mul(-1)
			}
		}
	}

	return mesh, nil
}

// isASCIISTL checks if the header suggests ASCII format
func isASCIISTL(header []byte) bool {
	// Look for "facet" or "endsolid" keywords
	s := strings.ToLower(string(header))
	return strings.Contains(s, "facet") || strings.Contains(s, "endsolid")
}

// loadSTLBinary loads a binary STL file
func loadSTLBinary(r io.Reader) (*Mesh, error) {
	// Read and discard header
	header := make([]byte, stlHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, errors.New("failed to read STL header")
	}

	// Read triangle count
	var triangleCount uint32
	if err := binary.Read(r, binary.LittleEndian, &triangleCount); err != nil {
		return nil, errors.New("failed to read triangle count")
	}

	// Sanity check
	if triangleCount > 10000000 {
		return nil, errors.New("STL file has too many triangles")
	}

	triangles := make([]Triangle, 0, triangleCount)

	// Read each triangle
	for i := uint32(0); i < triangleCount; i++ {
		var normal [3]float32
		var v1, v2, v3 [3]float32
		var attribute uint16

		// Read normal
		if err := binary.Read(r, binary.LittleEndian, &normal); err != nil {
			return nil, errors.New("failed to read triangle normal")
		}

		// Read vertices
		if err := binary.Read(r, binary.LittleEndian, &v1); err != nil {
			return nil, errors.New("failed to read vertex 1")
		}
		if err := binary.Read(r, binary.LittleEndian, &v2); err != nil {
			return nil, errors.New("failed to read vertex 2")
		}
		if err := binary.Read(r, binary.LittleEndian, &v3); err != nil {
			return nil, errors.New("failed to read vertex 3")
		}

		// Read attribute byte count (usually 0)
		if err := binary.Read(r, binary.LittleEndian, &attribute); err != nil {
			return nil, errors.New("failed to read attribute")
		}

		faceNormal := math.Vec3{
			X: float64(normal[0]),
			Y: float64(normal[1]),
			Z: float64(normal[2]),
		}

		// Normalize the face normal
		if faceNormal.Length() > 0 {
			faceNormal = faceNormal.Normalize()
		}

		tri := Triangle{
			Vertices: [3]Vertex{
				{
					Position: math.Vec3{X: float64(v1[0]), Y: float64(v1[1]), Z: float64(v1[2])},
					Normal:   faceNormal,
				},
				{
					Position: math.Vec3{X: float64(v2[0]), Y: float64(v2[1]), Z: float64(v2[2])},
					Normal:   faceNormal,
				},
				{
					Position: math.Vec3{X: float64(v3[0]), Y: float64(v3[1]), Z: float64(v3[2])},
					Normal:   faceNormal,
				},
			},
			FaceNormal: faceNormal,
		}

		triangles = append(triangles, tri)
	}

	mesh := &Mesh{
		Triangles: triangles,
	}

	return mesh, nil
}

// loadSTLASCII loads an ASCII STL file
func loadSTLASCII(r io.Reader) (*Mesh, error) {
	scanner := bufio.NewScanner(r)
	var triangles []Triangle
	var currentNormal math.Vec3
	var vertices []math.Vec3

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)

		if len(fields) == 0 {
			continue
		}

		switch strings.ToLower(fields[0]) {
		case "facet":
			// facet normal ni nj nk
			if len(fields) >= 5 && strings.ToLower(fields[1]) == "normal" {
				currentNormal = parseVec3(fields[2], fields[3], fields[4])
				if currentNormal.Length() > 0 {
					currentNormal = currentNormal.Normalize()
				}
			}
			vertices = nil

		case "vertex":
			// vertex x y z
			if len(fields) >= 4 {
				v := parseVec3(fields[1], fields[2], fields[3])
				vertices = append(vertices, v)
			}

		case "endfacet":
			// Create triangle from collected vertices
			if len(vertices) >= 3 {
				tri := Triangle{
					Vertices: [3]Vertex{
						{Position: vertices[0], Normal: currentNormal},
						{Position: vertices[1], Normal: currentNormal},
						{Position: vertices[2], Normal: currentNormal},
					},
					FaceNormal: currentNormal,
				}
				triangles = append(triangles, tri)
			}

		case "endsolid":
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	mesh := &Mesh{
		Triangles: triangles,
	}

	return mesh, nil
}

// parseVec3 parses three string fields into a Vec3
func parseVec3(x, y, z string) math.Vec3 {
	var fx, fy, fz float64
	fmt.Sscanf(x, "%f", &fx)
	fmt.Sscanf(y, "%f", &fy)
	fmt.Sscanf(z, "%f", &fz)
	return math.Vec3{X: fx, Y: fy, Z: fz}
}
