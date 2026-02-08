package tier1

import (
	"bytes"
	"fmt"
	"image"
)

// HalfBlockEncoder encodes an image.NRGBA as ANSI half-block characters.
// Each terminal cell represents 1x2 pixels using ▀ with fg=top, bg=bottom.
// Works in any 24-bit color terminal (xterm.js, conhost, SSH, etc).
// This is a lightweight preview of the full Tier 2 pipeline.
type HalfBlockEncoder struct {
	buf     bytes.Buffer
	prevFG  [3]uint8
	prevBG  [3]uint8
	hasPrev bool
}

// Encode converts an NRGBA image to ANSI half-block text.
// The image height should be even for best results.
func (e *HalfBlockEncoder) Encode(img *image.NRGBA) string {
	e.buf.Reset()
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	for y := 0; y < height-1; y += 2 {
		e.hasPrev = false

		for x := 0; x < width; x++ {
			// Top pixel = foreground, bottom pixel = background
			top := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			bot := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1)

			fg := [3]uint8{top.R, top.G, top.B}
			bg := [3]uint8{bot.R, bot.G, bot.B}

			// Only emit color changes when needed
			if !e.hasPrev || fg != e.prevFG {
				fmt.Fprintf(&e.buf, "\x1b[38;2;%d;%d;%dm", fg[0], fg[1], fg[2])
				e.prevFG = fg
			}
			if !e.hasPrev || bg != e.prevBG {
				fmt.Fprintf(&e.buf, "\x1b[48;2;%d;%d;%dm", bg[0], bg[1], bg[2])
				e.prevBG = bg
			}

			e.buf.WriteString("▀")
			e.hasPrev = true
		}

		// Reset colors and newline
		e.buf.WriteString("\x1b[0m\n")
	}

	return e.buf.String()
}

// EncodeWithCursor encodes with cursor positioning for animation.
func (e *HalfBlockEncoder) EncodeWithCursor(img *image.NRGBA, row, col int) string {
	e.buf.Reset()
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	for y := 0; y < height-1; y += 2 {
		// Position cursor for this row
		fmt.Fprintf(&e.buf, "\x1b[%d;%dH", row+y/2, col)
		e.hasPrev = false

		for x := 0; x < width; x++ {
			top := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			bot := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1)

			fg := [3]uint8{top.R, top.G, top.B}
			bg := [3]uint8{bot.R, bot.G, bot.B}

			if !e.hasPrev || fg != e.prevFG {
				fmt.Fprintf(&e.buf, "\x1b[38;2;%d;%d;%dm", fg[0], fg[1], fg[2])
				e.prevFG = fg
			}
			if !e.hasPrev || bg != e.prevBG {
				fmt.Fprintf(&e.buf, "\x1b[48;2;%d;%d;%dm", bg[0], bg[1], bg[2])
				e.prevBG = bg
			}

			e.buf.WriteString("▀")
			e.hasPrev = true
		}

		e.buf.WriteString("\x1b[0m")
	}

	return e.buf.String()
}

// InternalResolution returns the pixel resolution for ANSI half-block encoding.
// Each terminal cell is 1 pixel wide, 2 pixels tall.
func (e *HalfBlockEncoder) InternalResolution(termCols, termRows int) (int, int) {
	return termCols, termRows * 2
}
