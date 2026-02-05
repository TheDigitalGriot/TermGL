// Package gl provides 3D rendering pipeline for terminal graphics.
package gl

import (
	"github.com/charmbracelet/termgl/math"
)

// ClipPlane identifies a frustum clipping plane.
// Matches C enum ClipPlane from TermGL-C-Plus.
type ClipPlane int

const (
	ClipNear ClipPlane = iota
	ClipFar
	ClipLeft
	ClipRight
	ClipTop
	ClipBottom
)

// ClipVertex holds a clip-space vertex with UV coordinates.
// Matches the internal vertex structure used during clipping in C.
type ClipVertex struct {
	Position math.Vec4 // Clip space position (x, y, z, w)
	UV       [2]uint8  // Texture coordinates (u, v) 0-255
}

// ClipTriangle holds three vertices in clip space with UVs.
// Matches C TGLUVTriangle structure.
type ClipTriangle struct {
	Vertices [3]ClipVertex
}

// ClipPlaneDot computes the signed distance from a vertex to a clip plane.
// Positive = inside frustum, negative = outside frustum.
// Matches C itgl_clip_plane_dot function from termgl.c:1025-1044.
func ClipPlaneDot(v math.Vec4, plane ClipPlane) float64 {
	switch plane {
	case ClipLeft:
		return v.X + v.W // x >= -w
	case ClipRight:
		return -v.X + v.W // x <= w
	case ClipBottom:
		return v.Y + v.W // y >= -w
	case ClipTop:
		return -v.Y + v.W // y <= w
	case ClipNear:
		return v.Z + v.W // z >= -w
	case ClipFar:
		return -v.Z + v.W // z <= w
	}
	return 0
}

// mix performs linear interpolation: a + t*(b-a)
func mix(a, b, t float64) float64 {
	return a + t*(b-a)
}

// mixUint8 performs linear interpolation for uint8 values.
func mixUint8(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + t*(float64(b)-float64(a)))
}

// clipLine computes the intersection point between an edge and a clip plane.
// Matches C itgl_clip_line function from termgl.c:1012-1023.
func clipLine(dotIn float64, vIn ClipVertex, dotOut float64, vOut ClipVertex) ClipVertex {
	// Compute interpolation factor
	t := dotIn / (dotIn - dotOut)

	return ClipVertex{
		Position: math.Vec4{
			X: mix(vIn.Position.X, vOut.Position.X, t),
			Y: mix(vIn.Position.Y, vOut.Position.Y, t),
			Z: mix(vIn.Position.Z, vOut.Position.Z, t),
			W: mix(vIn.Position.W, vOut.Position.W, t),
		},
		UV: [2]uint8{
			mixUint8(vIn.UV[0], vOut.UV[0], t),
			mixUint8(vIn.UV[1], vOut.UV[1], t),
		},
	}
}

// ClipTriangleAgainstPlane clips a triangle against a single clip plane.
// Returns 0, 1, or 2 triangles using the Sutherland-Hodgman algorithm.
// Matches C itgl_clip_triangle_plane function from termgl.c:1046-1099.
func ClipTriangleAgainstPlane(tri ClipTriangle, plane ClipPlane) []ClipTriangle {
	// Compute signed distances for each vertex
	dots := [3]float64{
		ClipPlaneDot(tri.Vertices[0].Position, plane),
		ClipPlaneDot(tri.Vertices[1].Position, plane),
		ClipPlaneDot(tri.Vertices[2].Position, plane),
	}

	// Count vertices inside the plane (positive dot product)
	insideCount := 0
	for _, d := range dots {
		if d >= 0 {
			insideCount++
		}
	}

	switch insideCount {
	case 0:
		// All vertices outside - triangle is fully clipped
		return nil

	case 3:
		// All vertices inside - triangle is fully visible
		return []ClipTriangle{tri}

	case 1:
		// One vertex inside - produces 1 triangle
		// Find the inside vertex
		var insideIdx int
		for i, d := range dots {
			if d >= 0 {
				insideIdx = i
				break
			}
		}

		// Get vertices in winding order starting from inside vertex
		v0 := tri.Vertices[insideIdx]
		v1 := tri.Vertices[(insideIdx+1)%3]
		v2 := tri.Vertices[(insideIdx+2)%3]
		d0 := dots[insideIdx]
		d1 := dots[(insideIdx+1)%3]
		d2 := dots[(insideIdx+2)%3]

		// Compute intersection points
		newV1 := clipLine(d0, v0, d1, v1)
		newV2 := clipLine(d0, v0, d2, v2)

		return []ClipTriangle{
			{Vertices: [3]ClipVertex{v0, newV1, newV2}},
		}

	case 2:
		// Two vertices inside - produces 2 triangles (quad split)
		// Find the outside vertex
		var outsideIdx int
		for i, d := range dots {
			if d < 0 {
				outsideIdx = i
				break
			}
		}

		// Get vertices in winding order starting from outside vertex
		vOut := tri.Vertices[outsideIdx]
		v1 := tri.Vertices[(outsideIdx+1)%3]
		v2 := tri.Vertices[(outsideIdx+2)%3]
		dOut := dots[outsideIdx]
		d1 := dots[(outsideIdx+1)%3]
		d2 := dots[(outsideIdx+2)%3]

		// Compute intersection points on both edges from the outside vertex
		new1 := clipLine(d1, v1, dOut, vOut)
		new2 := clipLine(d2, v2, dOut, vOut)

		// Create two triangles from the quad
		return []ClipTriangle{
			{Vertices: [3]ClipVertex{v1, v2, new1}},
			{Vertices: [3]ClipVertex{new1, v2, new2}},
		}
	}

	return nil
}

// ClipTriangleAgainstFrustum clips a triangle against all 6 frustum planes.
// Returns 0 to N triangles after clipping.
// Matches the clipping loop in C tgl_triangle_3d from termgl.c:1124-1140.
func ClipTriangleAgainstFrustum(tri ClipTriangle) []ClipTriangle {
	triangles := []ClipTriangle{tri}

	// Clip against each plane in sequence
	for plane := ClipNear; plane <= ClipBottom; plane++ {
		if len(triangles) == 0 {
			return nil
		}

		var nextStage []ClipTriangle
		for _, t := range triangles {
			clipped := ClipTriangleAgainstPlane(t, plane)
			nextStage = append(nextStage, clipped...)
		}
		triangles = nextStage
	}

	return triangles
}

// IsTriangleVisible checks if any part of a triangle might be visible.
// This is a quick rejection test before expensive clipping.
func IsTriangleVisible(tri ClipTriangle) bool {
	// Check if all vertices are on the same side of any clip plane
	for plane := ClipNear; plane <= ClipBottom; plane++ {
		allOutside := true
		for _, v := range tri.Vertices {
			if ClipPlaneDot(v.Position, plane) >= 0 {
				allOutside = false
				break
			}
		}
		if allOutside {
			return false
		}
	}
	return true
}
