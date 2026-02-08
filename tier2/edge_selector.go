package tier2

import (
	"image"
	"image/color"
	"math"

	"github.com/fogleman/fauxgl"
)

// EdgeAwareSelector chooses characters that align with mesh edges.
// Architecture doc Section 5.5
type EdgeAwareSelector struct {
	blitter            Blitter
	depthMap           []float64
	normalMap          []fauxgl.Vector
	depthEdgeThreshold float64 // default: 0.05
	imgWidth           int
	imgHeight          int
}

// NewEdgeAwareSelector creates a new edge-aware character selector.
func NewEdgeAwareSelector(blitter Blitter, depthMap []float64, normalMap []fauxgl.Vector, imgWidth, imgHeight int) *EdgeAwareSelector {
	return &EdgeAwareSelector{
		blitter:            blitter,
		depthMap:           depthMap,
		normalMap:          normalMap,
		depthEdgeThreshold: 0.05,
		imgWidth:           imgWidth,
		imgHeight:          imgHeight,
	}
}

// SelectChar picks the best character and fg/bg colors for a cell.
// When depth discontinuities are detected, prefers diagonal/wedge characters.
func (s *EdgeAwareSelector) SelectChar(
	img *image.NRGBA,
	lumaPattern []float64,
	cellX, cellY int,
	cellW, cellH int,
) (char rune, fg, bg color.NRGBA) {

	// 1. Check for depth discontinuity in this cell
	hasEdge, edgeAngle := s.detectDepthEdge(cellX, cellY, cellW, cellH)

	if hasEdge {
		// 2a. Edge detected — use diagonal/wedge characters if angle is non-axis-aligned
		if edgeAngle > 10 && edgeAngle < 80 { // degrees from horizontal
			return s.selectDiagonalChar(img, lumaPattern, cellX, cellY, cellW, cellH, edgeAngle)
		}
	}

	// 2b. No edge or axis-aligned edge — use standard block character
	return s.selectBlockChar(img, lumaPattern, cellX, cellY, cellW, cellH)
}

// detectDepthEdge checks for depth discontinuities within a cell.
// Returns (hasEdge, angleInDegrees).
func (s *EdgeAwareSelector) detectDepthEdge(cellX, cellY, cellW, cellH int) (bool, float64) {
	if s.depthMap == nil || len(s.depthMap) == 0 {
		return false, 0
	}

	// Sample depth values at cell corners
	depths := make([]float64, 0, cellW*cellH)
	for sy := 0; sy < cellH; sy++ {
		for sx := 0; sx < cellW; sx++ {
			px := cellX*cellW + sx
			py := cellY*cellH + sy
			if px >= 0 && py >= 0 && px < s.imgWidth && py < s.imgHeight {
				idx := py*s.imgWidth + px
				if idx < len(s.depthMap) {
					depths = append(depths, s.depthMap[idx])
				}
			}
		}
	}

	if len(depths) < 2 {
		return false, 0
	}

	// Find min and max depth
	minDepth, maxDepth := depths[0], depths[0]
	for _, d := range depths {
		if d < minDepth {
			minDepth = d
		}
		if d > maxDepth {
			maxDepth = d
		}
	}

	// Check if depth range exceeds threshold
	depthRange := maxDepth - minDepth
	if depthRange < s.depthEdgeThreshold {
		return false, 0
	}

	// Estimate edge angle using depth gradient
	// Simplified: compute horizontal and vertical gradients
	var dx, dy float64
	if cellW >= 2 && cellH >= 2 {
		// Horizontal gradient (right - left)
		px1 := cellX*cellW
		py1 := cellY * cellH
		px2 := cellX*cellW + cellW - 1
		idx1 := py1*s.imgWidth + px1
		idx2 := py1*s.imgWidth + px2
		if idx1 < len(s.depthMap) && idx2 < len(s.depthMap) {
			dx = s.depthMap[idx2] - s.depthMap[idx1]
		}

		// Vertical gradient (bottom - top)
		py2 := cellY*cellH + cellH - 1
		idx1 = py1*s.imgWidth + px1
		idx2 = py2*s.imgWidth + px1
		if idx1 < len(s.depthMap) && idx2 < len(s.depthMap) {
			dy = s.depthMap[idx2] - s.depthMap[idx1]
		}
	}

	// Compute angle in degrees
	angleRad := math.Atan2(dy, dx)
	angleDeg := math.Abs(angleRad * 180.0 / math.Pi)

	return true, angleDeg
}

// selectDiagonalChar selects a diagonal or wedge character for edge-aligned rendering.
func (s *EdgeAwareSelector) selectDiagonalChar(
	img *image.NRGBA,
	lumaPattern []float64,
	cellX, cellY, cellW, cellH int,
	angle float64,
) (rune, color.NRGBA, color.NRGBA) {
	// For now, fall back to block character selection
	// Full diagonal character selection would use characters like /, \, ◢, ◣, ◤, ◥
	// This requires a more sophisticated pattern matching algorithm
	return s.selectBlockChar(img, lumaPattern, cellX, cellY, cellW, cellH)
}

// selectBlockChar selects a block character based on luminance thresholding.
func (s *EdgeAwareSelector) selectBlockChar(
	img *image.NRGBA,
	lumaPattern []float64,
	cellX, cellY, cellW, cellH int,
) (rune, color.NRGBA, color.NRGBA) {

	// Threshold luminance values to binary pattern
	threshold := medianLuma(lumaPattern)
	pattern := uint8(0)
	numBits := cellW * cellH

	for i := 0; i < numBits && i < len(lumaPattern); i++ {
		if lumaPattern[i] >= threshold {
			pattern |= 1 << uint(i)
		}
	}

	// Look up character from blitter table
	char := s.blitter.CharForPattern(pattern)

	// Collect pixel colors for this cell
	pixels := make([]color.NRGBA, 0, numBits)
	bounds := img.Bounds()
	for sy := 0; sy < cellH; sy++ {
		for sx := 0; sx < cellW; sx++ {
			px := bounds.Min.X + cellX*cellW + sx
			py := bounds.Min.Y + cellY*cellH + sy
			if px < bounds.Max.X && py < bounds.Max.Y {
				pixels = append(pixels, img.NRGBAAt(px, py))
			}
		}
	}

	// Compute fg/bg colors
	fg, bg := OptimizeCellColors(pixels, pattern, numBits)

	return char, fg, bg
}

// medianLuma finds the median luminance value.
func medianLuma(values []float64) float64 {
	if len(values) == 0 {
		return 0.5
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 && mid > 0 {
		return (sorted[mid-1] + sorted[mid]) / 2.0
	}
	return sorted[mid]
}
