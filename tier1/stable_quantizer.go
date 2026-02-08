package tier1

import (
	"image"
	"image/color"
)

// StablePaletteQuantizer maintains palette coherence across frames.
// Prevents flicker in animation by blending palettes between frames.
// Architecture doc Section 4.3
type StablePaletteQuantizer struct {
	prevPalette color.Palette
	adaptRate   float64          // 0.0 = locked palette, 1.0 = fully adaptive
	maxDrift    int              // Max palette entries that can change per frame
	inner       *OctreeQuantizer // Inner quantizer for palette generation
}

// NewStablePaletteQuantizer creates a quantizer that maintains palette coherence.
// Recommended defaults: adaptRate=0.3, maxDrift=32
func NewStablePaletteQuantizer(adaptRate float64, maxDrift int, inner *OctreeQuantizer) *StablePaletteQuantizer {
	return &StablePaletteQuantizer{
		adaptRate: adaptRate,
		maxDrift:  maxDrift,
		inner:     inner,
	}
}

// Quantize reduces an image to a paletted image with stable palette across frames.
func (q *StablePaletteQuantizer) Quantize(img *image.NRGBA, numColors int) *image.Paletted {
	// Generate optimal palette for this frame
	idealPalette := q.inner.buildPalette(img, numColors)

	// If we have a previous palette, blend toward the new one
	if q.prevPalette != nil && len(q.prevPalette) == len(idealPalette) {
		mergedPalette := blendPalettes(q.prevPalette, idealPalette, q.adaptRate, q.maxDrift)
		q.prevPalette = mergedPalette
		return q.inner.applyPalette(img, mergedPalette)
	}

	q.prevPalette = idealPalette
	return q.inner.applyPalette(img, idealPalette)
}

// Reset clears the previous palette state, forcing a full palette recomputation.
func (q *StablePaletteQuantizer) Reset() {
	q.prevPalette = nil
}
