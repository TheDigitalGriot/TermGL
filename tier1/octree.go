package tier1

import (
	"image"
	"image/color"
	"sort"
)

// OctreeQuantizer is fast and produces good results for real-time use.
// Architecture doc Section 4.3
type OctreeQuantizer struct {
	Dither     DitherMode
	ColorSpace ColorSpace
}

// octreeNode represents a node in the octree color quantization tree.
type octreeNode struct {
	isLeaf       bool
	refCount     int           // Number of pixels represented by this node
	redSum       int64         // Sum of red values
	greenSum     int64         // Sum of green values
	blueSum      int64         // Sum of blue values
	children     [8]*octreeNode
	paletteIndex int
}

// NewOctreeQuantizer creates a new octree quantizer with specified settings.
func NewOctreeQuantizer(dither DitherMode, space ColorSpace) *OctreeQuantizer {
	return &OctreeQuantizer{
		Dither:     dither,
		ColorSpace: space,
	}
}

// Quantize reduces an NRGBA image to a paletted image with the specified number of colors.
func (q *OctreeQuantizer) Quantize(img *image.NRGBA, numColors int) *image.Paletted {
	if numColors < 2 {
		numColors = 2
	}
	if numColors > 256 {
		numColors = 256
	}

	// Build octree from image
	root := &octreeNode{}
	leaves := make([]*octreeNode, 0, numColors)

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Insert all pixels into octree
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			q.insertColor(root, c, 0, &leaves)
		}
	}

	// Reduce tree to target number of colors
	for len(leaves) > numColors {
		// Find the leaf with the smallest reference count
		minIdx := 0
		minCount := leaves[0].refCount
		for i := 1; i < len(leaves); i++ {
			if leaves[i].refCount < minCount {
				minCount = leaves[i].refCount
				minIdx = i
			}
		}

		// Remove the leaf with smallest refCount by merging it with parent
		// In practice, we just remove it from the leaves list
		// The actual palette generation will handle averaging
		leaves = append(leaves[:minIdx], leaves[minIdx+1:]...)
	}

	// Build palette from remaining leaves
	palette := make(color.Palette, len(leaves))
	for i, leaf := range leaves {
		leaf.paletteIndex = i
		if leaf.refCount > 0 {
			r := uint8(leaf.redSum / int64(leaf.refCount))
			g := uint8(leaf.greenSum / int64(leaf.refCount))
			b := uint8(leaf.blueSum / int64(leaf.refCount))
			palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
		} else {
			palette[i] = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}

	// Create paletted image
	paletted := image.NewPaletted(bounds, palette)

	// Map pixels to palette with optional dithering
	switch q.Dither {
	case DitherFloydSteinberg:
		q.ditherFloydSteinberg(img, paletted, palette)
	case DitherOrdered4x4:
		q.ditherOrdered(img, paletted, palette, 4)
	case DitherOrdered8x8:
		q.ditherOrdered(img, paletted, palette, 8)
	default:
		q.ditherNone(img, paletted, palette)
	}

	return paletted
}

// insertColor inserts a color into the octree.
func (q *OctreeQuantizer) insertColor(node *octreeNode, c color.NRGBA, depth int, leaves *[]*octreeNode) {
	if depth >= 8 || node.isLeaf {
		// Reached maximum depth or existing leaf - accumulate color
		node.isLeaf = true
		node.refCount++
		node.redSum += int64(c.R)
		node.greenSum += int64(c.G)
		node.blueSum += int64(c.B)

		// Add to leaves list if not already there
		if node.refCount == 1 {
			*leaves = append(*leaves, node)
		}
		return
	}

	// Compute octree index based on color bits at current depth
	shift := 7 - depth
	idx := 0
	if (c.R>>shift)&1 == 1 {
		idx |= 4
	}
	if (c.G>>shift)&1 == 1 {
		idx |= 2
	}
	if (c.B>>shift)&1 == 1 {
		idx |= 1
	}

	// Create child node if it doesn't exist
	if node.children[idx] == nil {
		node.children[idx] = &octreeNode{}
	}

	// Recursively insert into child
	q.insertColor(node.children[idx], c, depth+1, leaves)
}

// ditherNone performs no dithering - direct nearest-color mapping.
func (q *OctreeQuantizer) ditherNone(src *image.NRGBA, dst *image.Paletted, palette color.Palette) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			idx := q.nearestColor(c, palette)
			dst.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, uint8(idx))
		}
	}
}

// ditherFloydSteinberg performs Floyd-Steinberg error diffusion dithering.
func (q *OctreeQuantizer) ditherFloydSteinberg(src *image.NRGBA, dst *image.Paletted, palette color.Palette) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Create error accumulation buffer
	errors := make([][]struct{ r, g, b float64 }, height)
	for y := range errors {
		errors[y] = make([]struct{ r, g, b float64 }, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get original color and add accumulated error
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			r := float64(c.R) + errors[y][x].r
			g := float64(c.G) + errors[y][x].g
			b := float64(c.B) + errors[y][x].b

			// Clamp to valid range
			r = clamp(r, 0, 255)
			g = clamp(g, 0, 255)
			b = clamp(b, 0, 255)

			newColor := color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}

			// Find nearest palette color
			idx := q.nearestColor(newColor, palette)
			dst.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, uint8(idx))

			// Compute quantization error
			palColor := palette[idx].(color.NRGBA)
			errR := r - float64(palColor.R)
			errG := g - float64(palColor.G)
			errB := b - float64(palColor.B)

			// Distribute error to neighboring pixels
			// Floyd-Steinberg distribution:
			//     X   7/16
			// 3/16 5/16 1/16

			if x+1 < width {
				errors[y][x+1].r += errR * 7.0 / 16.0
				errors[y][x+1].g += errG * 7.0 / 16.0
				errors[y][x+1].b += errB * 7.0 / 16.0
			}

			if y+1 < height {
				if x > 0 {
					errors[y+1][x-1].r += errR * 3.0 / 16.0
					errors[y+1][x-1].g += errG * 3.0 / 16.0
					errors[y+1][x-1].b += errB * 3.0 / 16.0
				}

				errors[y+1][x].r += errR * 5.0 / 16.0
				errors[y+1][x].g += errG * 5.0 / 16.0
				errors[y+1][x].b += errB * 5.0 / 16.0

				if x+1 < width {
					errors[y+1][x+1].r += errR * 1.0 / 16.0
					errors[y+1][x+1].g += errG * 1.0 / 16.0
					errors[y+1][x+1].b += errB * 1.0 / 16.0
				}
			}
		}
	}
}

// ditherOrdered performs ordered (Bayer) dithering.
func (q *OctreeQuantizer) ditherOrdered(src *image.NRGBA, dst *image.Paletted, palette color.Palette, matrixSize int) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	var matrix [][]float64
	if matrixSize == 4 {
		matrix = make([][]float64, 4)
		for i := range matrix {
			matrix[i] = bayer4x4[i][:]
		}
	} else {
		matrix = make([][]float64, 8)
		for i := range matrix {
			matrix[i] = bayer8x8[i][:]
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)

			// Apply Bayer threshold
			threshold := matrix[y%matrixSize][x%matrixSize]
			r := float64(c.R) + (threshold-0.5)*32.0
			g := float64(c.G) + (threshold-0.5)*32.0
			b := float64(c.B) + (threshold-0.5)*32.0

			r = clamp(r, 0, 255)
			g = clamp(g, 0, 255)
			b = clamp(b, 0, 255)

			newColor := color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
			idx := q.nearestColor(newColor, palette)
			dst.SetColorIndex(bounds.Min.X+x, bounds.Min.Y+y, uint8(idx))
		}
	}
}

// nearestColor finds the index of the nearest color in the palette.
func (q *OctreeQuantizer) nearestColor(c color.NRGBA, palette color.Palette) int {
	minDist := -1.0
	minIdx := 0

	for i, palColor := range palette {
		dist := colorDistance(c, palColor.(color.NRGBA), q.ColorSpace)
		if minDist < 0 || dist < minDist {
			minDist = dist
			minIdx = i
		}
	}

	return minIdx
}

// buildPalette builds a palette from the image (exposed for StablePaletteQuantizer).
func (q *OctreeQuantizer) buildPalette(img *image.NRGBA, numColors int) color.Palette {
	if numColors < 2 {
		numColors = 2
	}
	if numColors > 256 {
		numColors = 256
	}

	// Build octree from image
	root := &octreeNode{}
	leaves := make([]*octreeNode, 0, numColors)

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Insert all pixels into octree
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			q.insertColor(root, c, 0, &leaves)
		}
	}

	// Reduce tree to target number of colors
	for len(leaves) > numColors {
		// Find the leaf with the smallest reference count
		minIdx := 0
		minCount := leaves[0].refCount
		for i := 1; i < len(leaves); i++ {
			if leaves[i].refCount < minCount {
				minCount = leaves[i].refCount
				minIdx = i
			}
		}

		leaves = append(leaves[:minIdx], leaves[minIdx+1:]...)
	}

	// Build palette from remaining leaves
	palette := make(color.Palette, len(leaves))
	for i, leaf := range leaves {
		if leaf.refCount > 0 {
			r := uint8(leaf.redSum / int64(leaf.refCount))
			g := uint8(leaf.greenSum / int64(leaf.refCount))
			b := uint8(leaf.blueSum / int64(leaf.refCount))
			palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
		} else {
			palette[i] = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		}
	}

	return palette
}

// applyPalette applies a given palette to an image with dithering.
func (q *OctreeQuantizer) applyPalette(img *image.NRGBA, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, palette)

	switch q.Dither {
	case DitherFloydSteinberg:
		q.ditherFloydSteinberg(img, paletted, palette)
	case DitherOrdered4x4:
		q.ditherOrdered(img, paletted, palette, 4)
	case DitherOrdered8x8:
		q.ditherOrdered(img, paletted, palette, 8)
	default:
		q.ditherNone(img, paletted, palette)
	}

	return paletted
}

// clamp restricts a value to the range [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// blendPalettes blends two palettes together based on adapt rate and max drift.
func blendPalettes(oldPal, newPal color.Palette, adaptRate float64, maxDrift int) color.Palette {
	if len(oldPal) != len(newPal) {
		return newPal
	}

	result := make(color.Palette, len(oldPal))

	// Match new palette colors to old palette colors
	type match struct {
		oldIdx int
		newIdx int
		dist   float64
	}

	matches := make([]match, 0, len(oldPal))

	// Find best matches between old and new palettes
	used := make([]bool, len(newPal))
	for oldIdx := range oldPal {
		bestNewIdx := -1
		bestDist := -1.0

		for newIdx := range newPal {
			if used[newIdx] {
				continue
			}
			dist := colorDistanceRGB(oldPal[oldIdx].(color.NRGBA), newPal[newIdx].(color.NRGBA))
			if bestNewIdx < 0 || dist < bestDist {
				bestDist = dist
				bestNewIdx = newIdx
			}
		}

		if bestNewIdx >= 0 {
			matches = append(matches, match{oldIdx: oldIdx, newIdx: bestNewIdx, dist: bestDist})
			used[bestNewIdx] = true
		}
	}

	// Sort matches by distance (smallest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].dist < matches[j].dist
	})

	// Apply palette drift limit
	drifted := 0
	for _, m := range matches {
		oldColor := oldPal[m.oldIdx].(color.NRGBA)
		newColor := newPal[m.newIdx].(color.NRGBA)

		// If we haven't exceeded max drift, blend colors
		if drifted < maxDrift || m.dist < 1000 { // Always blend very close matches
			// Blend old and new colors
			r := uint8(float64(oldColor.R)*(1-adaptRate) + float64(newColor.R)*adaptRate)
			g := uint8(float64(oldColor.G)*(1-adaptRate) + float64(newColor.G)*adaptRate)
			b := uint8(float64(oldColor.B)*(1-adaptRate) + float64(newColor.B)*adaptRate)
			result[m.oldIdx] = color.NRGBA{R: r, G: g, B: b, A: 255}
			drifted++
		} else {
			// Keep old color
			result[m.oldIdx] = oldColor
		}
	}

	return result
}
