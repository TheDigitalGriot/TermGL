package tier1

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"strconv"
)

// SixelEncoder converts paletted images to Sixel escape sequences.
// Architecture doc Section 4.4
type SixelEncoder struct {
	MaxColors int  // Caps the palette size (default: 256)
	RLE       bool // Enable run-length encoding (default: true)

	// Pre-computed strings for performance (inspired by impulse demo)
	colorSelectors [256]string // "#<idx>" strings
	rleCountStrs   [256]string // "!<count>" strings

	// Reusable buffer across frames
	buf bytes.Buffer

	// Track colors used per band to skip unused
	bandColorUsed [256]bool
}

// NewSixelEncoder creates a new Sixel encoder with the specified settings.
func NewSixelEncoder(maxColors int, rle bool) *SixelEncoder {
	if maxColors <= 0 || maxColors > 256 {
		maxColors = 256
	}

	e := &SixelEncoder{
		MaxColors: maxColors,
		RLE:       rle,
	}

	// Pre-compute color selector strings
	for i := 0; i < 256; i++ {
		e.colorSelectors[i] = "#" + strconv.Itoa(i)
		e.rleCountStrs[i] = "!" + strconv.Itoa(i)
	}

	return e
}

// Encode writes a paletted image as Sixel escape sequences to w.
func (e *SixelEncoder) Encode(w io.Writer, img *image.Paletted) error {
	e.buf.Reset()
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	palette := img.Palette

	numColors := len(palette)
	if numColors > e.MaxColors {
		numColors = e.MaxColors
	}

	// DCS introducer: ESC P 0;1;0 q
	// Params: aspect=0, background=1 (transparent), grid=0
	e.buf.WriteString("\x1bP0;1;0q")

	// Emit palette definitions
	// Format: #<index>;2;<r%>;<g%>;<b%>
	// RGB components are 0-100 (percentage)
	for i := 0; i < numColors; i++ {
		r, g, b, _ := palette[i].RGBA()
		// Convert from 0-65535 to 0-100
		rPct := r * 100 / 65535
		gPct := g * 100 / 65535
		bPct := b * 100 / 65535
		fmt.Fprintf(&e.buf, "#%d;2;%d;%d;%d", i, rPct, gPct, bPct)
	}

	// Encode image data in 6-row bands
	for bandY := 0; bandY < height; bandY += 6 {
		bandHeight := 6
		if bandY+6 > height {
			bandHeight = height - bandY
		}

		// Determine which colors are used in this band
		for i := range e.bandColorUsed {
			e.bandColorUsed[i] = false
		}
		for y := bandY; y < bandY+bandHeight; y++ {
			for x := bounds.Min.X; x < bounds.Min.X+width; x++ {
				idx := img.ColorIndexAt(x, y+bounds.Min.Y)
				if int(idx) < numColors {
					e.bandColorUsed[idx] = true
				}
			}
		}

		firstColor := true
		for colorIdx := 0; colorIdx < numColors; colorIdx++ {
			if !e.bandColorUsed[colorIdx] {
				continue
			}

			// Select this color
			e.buf.WriteString(e.colorSelectors[colorIdx])

			// Encode columns for this color in this band
			if e.RLE {
				e.encodeBandColorRLE(img, bounds, width, bandY, bandHeight, colorIdx)
			} else {
				e.encodeBandColorRaw(img, bounds, width, bandY, bandHeight, colorIdx)
			}

			// Carriage return within band (go back to column 0 for next color)
			e.buf.WriteByte('$')
			firstColor = false
		}

		// Move to next 6-row band
		if bandY+6 < height {
			if firstColor {
				// Empty band — still need to move down
				e.buf.WriteByte('-')
			} else {
				// Replace last '$' with '-' (newline instead of carriage return)
				// The last byte is '$', replace it
				data := e.buf.Bytes()
				data[len(data)-1] = '-'
			}
		}
	}

	// String Terminator: ESC backslash
	e.buf.WriteString("\x1b\\")

	_, err := w.Write(e.buf.Bytes())
	return err
}

// encodeBandColorRLE encodes one color's contribution to a band with RLE compression.
func (e *SixelEncoder) encodeBandColorRLE(img *image.Paletted, bounds image.Rectangle, width, bandY, bandHeight, colorIdx int) {
	prevChar := byte(0)
	runLength := 0

	for x := 0; x < width; x++ {
		// Build the 6-bit sixel value for this column
		sixelVal := byte(0)
		for bit := 0; bit < bandHeight; bit++ {
			py := bandY + bit + bounds.Min.Y
			px := x + bounds.Min.X
			if img.ColorIndexAt(px, py) == uint8(colorIdx) {
				sixelVal |= 1 << uint(bit)
			}
		}

		// Sixel characters are value + 63 (0x3F)
		sixelChar := sixelVal + 63

		if sixelChar == prevChar && runLength > 0 {
			runLength++
		} else {
			e.flushRun(prevChar, runLength)
			prevChar = sixelChar
			runLength = 1
		}
	}
	e.flushRun(prevChar, runLength)
}

// encodeBandColorRaw encodes one color's contribution to a band without RLE.
func (e *SixelEncoder) encodeBandColorRaw(img *image.Paletted, bounds image.Rectangle, width, bandY, bandHeight, colorIdx int) {
	for x := 0; x < width; x++ {
		sixelVal := byte(0)
		for bit := 0; bit < bandHeight; bit++ {
			py := bandY + bit + bounds.Min.Y
			px := x + bounds.Min.X
			if img.ColorIndexAt(px, py) == uint8(colorIdx) {
				sixelVal |= 1 << uint(bit)
			}
		}
		e.buf.WriteByte(sixelVal + 63)
	}
}

// flushRun writes a run of identical sixel characters.
func (e *SixelEncoder) flushRun(char byte, count int) {
	if count <= 0 {
		return
	}

	switch count {
	case 1:
		e.buf.WriteByte(char)
	case 2:
		e.buf.WriteByte(char)
		e.buf.WriteByte(char)
	case 3:
		e.buf.WriteByte(char)
		e.buf.WriteByte(char)
		e.buf.WriteByte(char)
	default:
		// RLE: !<count><char>
		if count < 256 {
			e.buf.WriteString(e.rleCountStrs[count])
		} else {
			fmt.Fprintf(&e.buf, "!%d", count)
		}
		e.buf.WriteByte(char)
	}
}

// EncodedSize returns the approximate encoded size of the last frame.
func (e *SixelEncoder) EncodedSize() int {
	return e.buf.Len()
}
