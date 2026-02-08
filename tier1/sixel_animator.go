package tier1

import (
	"fmt"
	"image"
	"io"
)

// SixelAnimator manages frame delivery for real-time Sixel animation.
// Handles cursor positioning and quantization pipeline.
// Architecture doc Section 4.5
type SixelAnimator struct {
	Encoder   *SixelEncoder
	Quantizer Quantizer // StablePaletteQuantizer or FixedPaletteQuantizer
	Writer    io.Writer

	// Cursor positioning
	OriginRow int // Row where the image starts (1-based)
	OriginCol int // Column where the image starts (1-based)
	ImgRows   int // Number of terminal rows the image occupies
}

// NewSixelAnimator creates a new animator for real-time Sixel animation.
func NewSixelAnimator(encoder *SixelEncoder, quantizer Quantizer, writer io.Writer) *SixelAnimator {
	return &SixelAnimator{
		Encoder:   encoder,
		Quantizer: quantizer,
		Writer:    writer,
		OriginRow: 1,
		OriginCol: 1,
	}
}

// DrawFrame renders and displays a single animation frame.
func (a *SixelAnimator) DrawFrame(img *image.NRGBA) error {
	// Quantize to paletted image
	paletted := a.Quantizer.Quantize(img, a.Encoder.MaxColors)

	// Move cursor to image origin
	// CSI <row>;<col>H — cursor position
	fmt.Fprintf(a.Writer, "\x1b[%d;%dH", a.OriginRow, a.OriginCol)

	// Encode and write Sixel data
	return a.Encoder.Encode(a.Writer, paletted)
}

// Setup calculates image dimensions and prepares the display area.
// cellHeight is the height of one terminal cell in pixels.
func (a *SixelAnimator) Setup(cellHeight, imgHeight int) {
	if cellHeight <= 0 {
		cellHeight = 20 // Reasonable default
	}

	// Calculate how many terminal rows the image will occupy
	a.ImgRows = (imgHeight + cellHeight - 1) / cellHeight

	// Reserve space by scrolling
	for i := 0; i < a.ImgRows; i++ {
		fmt.Fprint(a.Writer, "\n")
	}

	// Move cursor back up to the start of the reserved area
	fmt.Fprintf(a.Writer, "\x1b[%dA", a.ImgRows)

	// Save current position as origin (default to row 1, col 1)
	// In practice, the caller should set OriginRow/OriginCol
	// after determining the actual cursor position
}

// SetOrigin sets the cursor origin for frame rendering.
func (a *SixelAnimator) SetOrigin(row, col int) {
	a.OriginRow = row
	a.OriginCol = col
}
