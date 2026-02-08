package render

import (
	"image"
)

// Encoder is the interface shared by both output tiers.
// Architecture doc Section 6.1
type Encoder interface {
	// Init sets up the encoder for the given terminal capabilities.
	Init(caps TerminalCaps) error

	// Encode converts a rendered frame to terminal output.
	// Returns the string to write to stdout.
	Encode(frame *image.NRGBA) string

	// InternalResolution returns the pixel resolution the rasterizer
	// should render at for this encoder.
	InternalResolution(termCols, termRows int) (pixelW, pixelH int)
}

// AuxEncoder extends Encoder with auxiliary buffer support.
// Tier 2 implements this for edge-aware character selection.
type AuxEncoder interface {
	Encoder

	// EncodeWithAux converts a rendered frame using auxiliary depth/normal buffers.
	EncodeWithAux(frame *image.NRGBA, aux *AuxBuffers) string
}
