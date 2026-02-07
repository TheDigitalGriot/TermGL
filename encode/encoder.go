// Package encode provides terminal pixel encoders for the adaptive framebuffer.
// Each encoder converts image.RGBA data to ANSI escape sequences optimized
// for a specific terminal capability level.
package encode

import (
	"image"

	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/framebuffer"
)

// Encoder converts framebuffer pixels to terminal output
type Encoder interface {
	// Level returns the encoder's capability level
	Level() detect.EncoderLevel

	// Encode converts the framebuffer to ANSI escape sequences.
	// Only the dirty region needs to be encoded.
	// Returns the bytes to write to stdout.
	Encode(fb *framebuffer.Framebuffer) []byte

	// Reset clears internal state (e.g., previous frame cache)
	Reset()

	// GridSize returns the terminal grid size this encoder uses
	GridSize() image.Point
}

// colorDistance calculates squared Euclidean distance between two colors.
// Used for delta detection with perceptual thresholding.
func colorDistance(r1, g1, b1, r2, g2, b2 uint8) int {
	dr := int(r1) - int(r2)
	dg := int(g1) - int(g2)
	db := int(b1) - int(b2)
	return dr*dr + dg*dg + db*db
}

// PerceptuallyEqual returns true if two colors are visually indistinguishable.
// Uses weighted Euclidean distance approximating human perception.
func PerceptuallyEqual(r1, g1, b1, r2, g2, b2 uint8, threshold float64) bool {
	dr := float64(r1) - float64(r2)
	dg := float64(g1) - float64(g2)
	db := float64(b1) - float64(b2)
	// Weighted by human perception (rec. 601 luma coefficients)
	dist := 0.299*dr*dr + 0.587*dg*dg + 0.114*db*db
	return dist < threshold*threshold
}
