package tier2

import (
	"github.com/charmbracelet/termgl/tier1"
)

// BayerDither applies ordered dithering to luminance values.
// Bayer 4x4 for animation (no crawling artifacts between frames).
// Bayer 8x8 for smoother gradients.
// Architecture doc Section 5.2
func BayerDither(y []float64, width, height int, mode tier1.DitherMode) {
	if mode == tier1.DitherNone || mode == tier1.DitherFloydSteinberg {
		return // No ordered dithering for these modes
	}

	// Use the Bayer matrices from tier1
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			idx := py*width + px

			var threshold float64
			if mode == tier1.DitherOrdered4x4 {
				// Bayer 4x4
				threshold = tier1.GetBayer4x4(px, py)
			} else {
				// Bayer 8x8
				threshold = tier1.GetBayer8x8(px, py)
			}

			// Apply dithering: add threshold - 0.5 to shift the quantization point
			y[idx] += (threshold - 0.5) / 16.0
		}
	}
}
