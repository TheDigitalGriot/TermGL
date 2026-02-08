package tier2

import (
	"image/color"
	"math"
)

// OptimizeCellColors finds the best fg/bg color pair for a cell.
// Uses simplified clustering: split colors based on luminance.
// For performance, we use a simple luminance-based split rather than full PCA.
// Architecture doc Section 5.2
func OptimizeCellColors(pixels []color.NRGBA, pattern uint8, numBits int) (fg, bg color.NRGBA) {
	if len(pixels) == 0 {
		return color.NRGBA{}, color.NRGBA{}
	}

	// Separate pixels into "on" and "off" groups based on pattern
	var fgPixels, bgPixels []color.NRGBA

	for i, pixel := range pixels {
		if i >= numBits {
			break
		}
		if pattern&(1<<uint(i)) != 0 {
			fgPixels = append(fgPixels, pixel)
		} else {
			bgPixels = append(bgPixels, pixel)
		}
	}

	// Compute average color for each group
	fg = averageColor(fgPixels)
	bg = averageColor(bgPixels)

	return fg, bg
}

// averageColor computes the mean color of a set of pixels.
func averageColor(pixels []color.NRGBA) color.NRGBA {
	if len(pixels) == 0 {
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}

	var sumR, sumG, sumB uint32
	for _, p := range pixels {
		sumR += uint32(p.R)
		sumG += uint32(p.G)
		sumB += uint32(p.B)
	}

	n := uint32(len(pixels))
	return color.NRGBA{
		R: uint8(sumR / n),
		G: uint8(sumG / n),
		B: uint8(sumB / n),
		A: 255,
	}
}

// OptimizeCellColorsPCA performs PCA-based color optimization.
// This is the full algorithm from the architecture doc, but more expensive.
// Use this when quality is more important than performance.
func OptimizeCellColorsPCA(pixels []color.NRGBA, pattern uint8, numBits int) (fg, bg color.NRGBA) {
	if len(pixels) == 0 {
		return color.NRGBA{}, color.NRGBA{}
	}

	// Convert to float64 for PCA
	type Vec3 struct{ r, g, b float64 }
	var points []Vec3
	for _, p := range pixels {
		points = append(points, Vec3{
			r: float64(p.R) / 255.0,
			g: float64(p.G) / 255.0,
			b: float64(p.B) / 255.0,
		})
	}

	// Compute centroid
	var centroid Vec3
	for _, p := range points {
		centroid.r += p.r
		centroid.g += p.g
		centroid.b += p.b
	}
	centroid.r /= float64(len(points))
	centroid.g /= float64(len(points))
	centroid.b /= float64(len(points))

	// Compute covariance matrix (simplified - just find dominant axis)
	var covRR, covGG, covBB float64
	for _, p := range points {
		dr := p.r - centroid.r
		dg := p.g - centroid.g
		db := p.b - centroid.b
		covRR += dr * dr
		covGG += dg * dg
		covBB += db * db
	}

	// Find dominant axis (largest variance)
	var projectAxis Vec3
	if covRR >= covGG && covRR >= covBB {
		projectAxis = Vec3{1, 0, 0}
	} else if covGG >= covBB {
		projectAxis = Vec3{0, 1, 0}
	} else {
		projectAxis = Vec3{0, 0, 1}
	}

	// Project points onto dominant axis
	projections := make([]float64, len(points))
	for i, p := range points {
		projections[i] = p.r*projectAxis.r + p.g*projectAxis.g + p.b*projectAxis.b
	}

	// Find median projection value
	median := medianFloat64(projections)

	// Split into two clusters based on median
	var cluster1, cluster2 []color.NRGBA
	for i, proj := range projections {
		if proj < median {
			cluster1 = append(cluster1, pixels[i])
		} else {
			cluster2 = append(cluster2, pixels[i])
		}
	}

	// Assign clusters to fg/bg based on pattern
	// For simplicity, use the larger cluster for the more common bits in the pattern
	onCount := countBits(pattern, numBits)
	if onCount > numBits/2 {
		fg = averageColor(cluster2)
		bg = averageColor(cluster1)
	} else {
		fg = averageColor(cluster1)
		bg = averageColor(cluster2)
	}

	return fg, bg
}

// countBits counts the number of set bits in a pattern.
func countBits(pattern uint8, numBits int) int {
	count := 0
	for i := 0; i < numBits; i++ {
		if pattern&(1<<uint(i)) != 0 {
			count++
		}
	}
	return count
}

// medianFloat64 finds the median value in a slice (modifies the slice).
func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple selection - for production, use a proper nth_element algorithm
	// For now, just sort and pick middle
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Bubble sort (simple, good enough for small arrays)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2.0
	}
	return sorted[mid]
}

// luminance computes the perceptual luminance of a color (BT.601).
func luminance(c color.NRGBA) float64 {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	return 0.299*r + 0.587*g + 0.114*b
}

// perceptualDistance computes approximate perceptual color distance.
// For performance, uses weighted Euclidean in RGB rather than full DIN99d.
func perceptualDistance(c1, c2 color.NRGBA) float64 {
	// Redmean approximation (faster than CIELAB, better than RGB)
	rmean := (float64(c1.R) + float64(c2.R)) / 2.0
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)

	return math.Sqrt((2.0+rmean/256.0)*dr*dr + 4.0*dg*dg + (2.0+(255.0-rmean)/256.0)*db*db)
}
