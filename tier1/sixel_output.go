package tier1

import (
	"bytes"
	"image"

	"github.com/charmbracelet/termgl/render"
)

// SixelOutput implements render.Encoder for Tier 1 (Sixel).
// Architecture doc Section 6.1
type SixelOutput struct {
	encoder   *SixelEncoder
	quantizer Quantizer
	animator  *SixelAnimator
	cellW     int
	cellH     int
	buf       bytes.Buffer
}

// NewSixelOutput creates a new SixelOutput encoder.
func NewSixelOutput(quantizer Quantizer, maxColors int, rle bool) *SixelOutput {
	encoder := NewSixelEncoder(maxColors, rle)
	return &SixelOutput{
		encoder:   encoder,
		quantizer: quantizer,
		cellW:     8,  // Default cell width in pixels
		cellH:     16, // Default cell height in pixels
	}
}

// Init sets up the encoder for the given terminal capabilities.
// Implements render.Encoder.
func (s *SixelOutput) Init(caps render.TerminalCaps) error {
	if caps.CellWidth > 0 {
		s.cellW = caps.CellWidth
	}
	if caps.CellHeight > 0 {
		s.cellH = caps.CellHeight
	}
	if caps.SixelMaxColors > 0 && caps.SixelMaxColors < s.encoder.MaxColors {
		s.encoder.MaxColors = caps.SixelMaxColors
	}

	// Create animator for frame delivery
	s.animator = NewSixelAnimator(s.encoder, s.quantizer, &s.buf)
	return nil
}

// Encode converts a rendered frame to Sixel escape sequences.
// Returns the string to write to stdout.
// Implements render.Encoder.
func (s *SixelOutput) Encode(frame *image.NRGBA) string {
	s.buf.Reset()

	// Quantize
	paletted := s.quantizer.Quantize(frame, s.encoder.MaxColors)

	// Encode to Sixel
	_ = s.encoder.Encode(&s.buf, paletted)

	return s.buf.String()
}

// EncodeWithCursor converts a rendered frame to Sixel escape sequences
// with cursor positioning for animation.
func (s *SixelOutput) EncodeWithCursor(frame *image.NRGBA, row, col int) string {
	s.buf.Reset()

	// Move cursor to image origin
	s.buf.WriteString("\x1b[")
	s.buf.WriteString(itoa(row))
	s.buf.WriteByte(';')
	s.buf.WriteString(itoa(col))
	s.buf.WriteByte('H')

	// Quantize
	paletted := s.quantizer.Quantize(frame, s.encoder.MaxColors)

	// Encode to Sixel
	_ = s.encoder.Encode(&s.buf, paletted)

	return s.buf.String()
}

// InternalResolution returns the pixel resolution the rasterizer should render at.
// Sixel renders at pixel resolution based on cell dimensions.
// Implements render.Encoder.
func (s *SixelOutput) InternalResolution(termCols, termRows int) (int, int) {
	return termCols * s.cellW, termRows * s.cellH
}

// fast integer to string without fmt
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
