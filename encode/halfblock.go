package encode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/framebuffer"
)

// HalfBlockCell represents one terminal cell with two vertical pixels
type HalfBlockCell struct {
	Top    color.RGBA // foreground color (▀ character)
	Bottom color.RGBA // background color
}

// HalfBlockEncoder converts image.RGBA to half-block ANSI output.
// Each terminal cell displays ▀ with fg=top pixel, bg=bottom pixel.
// This gives 2 vertical pixels per cell with independent 24-bit colors.
type HalfBlockEncoder struct {
	buf       bytes.Buffer
	prevFrame []HalfBlockCell
	gridWidth int
	gridHeight int

	// Track last emitted colors for RLE optimization
	lastFG color.RGBA
	lastBG color.RGBA

	// Perceptual threshold for skipping nearly-identical cells
	threshold float64
}

// NewHalfBlockEncoder creates a new half-block encoder for the given grid size
func NewHalfBlockEncoder(gridWidth, gridHeight int) *HalfBlockEncoder {
	return &HalfBlockEncoder{
		gridWidth:  gridWidth,
		gridHeight: gridHeight,
		prevFrame:  make([]HalfBlockCell, gridWidth*gridHeight),
		threshold:  2.0, // ΔE < 2 is imperceptible
	}
}

// Level returns LevelHalfBlock
func (e *HalfBlockEncoder) Level() detect.EncoderLevel {
	return detect.LevelHalfBlock
}

// GridSize returns the terminal grid dimensions
func (e *HalfBlockEncoder) GridSize() image.Point {
	return image.Point{X: e.gridWidth, Y: e.gridHeight}
}

// Reset clears the previous frame cache, forcing a full redraw
func (e *HalfBlockEncoder) Reset() {
	for i := range e.prevFrame {
		e.prevFrame[i] = HalfBlockCell{}
	}
	e.lastFG = color.RGBA{}
	e.lastBG = color.RGBA{}
}

// SetThreshold sets the perceptual difference threshold.
// Colors with squared distance below this are treated as identical.
func (e *HalfBlockEncoder) SetThreshold(t float64) {
	e.threshold = t
}

// Encode converts the framebuffer to ANSI escape sequences.
// Uses delta encoding to skip unchanged cells.
// Uses color RLE to avoid re-emitting identical colors.
func (e *HalfBlockEncoder) Encode(fb *framebuffer.Framebuffer) []byte {
	e.buf.Reset()

	// Resize prevFrame if grid size changed
	if len(e.prevFrame) != e.gridWidth*e.gridHeight {
		e.prevFrame = make([]HalfBlockCell, e.gridWidth*e.gridHeight)
		e.lastFG = color.RGBA{}
		e.lastBG = color.RGBA{}
	}

	// Track if we're on a consecutive run (for cursor optimization)
	lastRow := -1
	lastCol := -1

	for row := 0; row < e.gridHeight; row++ {
		for col := 0; col < e.gridWidth; col++ {
			// Map grid position to framebuffer pixels
			// Each grid row represents 2 vertical pixels
			topY := row * 2
			botY := row*2 + 1

			// Get pixel colors (clamp to bounds)
			var top, bot color.RGBA
			if topY < fb.Height && col < fb.Width {
				top = fb.Pixels.RGBAAt(col, topY)
			}
			if botY < fb.Height && col < fb.Width {
				bot = fb.Pixels.RGBAAt(col, botY)
			}

			cell := HalfBlockCell{Top: top, Bottom: bot}
			idx := row*e.gridWidth + col

			// Delta check: skip if cell unchanged (with perceptual threshold)
			prev := e.prevFrame[idx]
			if e.cellsEqual(cell, prev) {
				continue
			}

			// Update previous frame
			e.prevFrame[idx] = cell

			// Emit cursor position if not consecutive
			if row != lastRow || col != lastCol+1 {
				// ANSI cursor position is 1-indexed
				fmt.Fprintf(&e.buf, "\x1b[%d;%dH", row+1, col+1)
			}

			// Emit foreground color if changed (top pixel)
			if top != e.lastFG {
				fmt.Fprintf(&e.buf, "\x1b[38;2;%d;%d;%dm", top.R, top.G, top.B)
				e.lastFG = top
			}

			// Emit background color if changed (bottom pixel)
			if bot != e.lastBG {
				fmt.Fprintf(&e.buf, "\x1b[48;2;%d;%d;%dm", bot.R, bot.G, bot.B)
				e.lastBG = bot
			}

			// Emit the half-block character
			e.buf.WriteString("▀")

			lastRow = row
			lastCol = col
		}
	}

	// Reset colors at end of frame
	if e.buf.Len() > 0 {
		e.buf.WriteString("\x1b[0m")
	}

	return e.buf.Bytes()
}

// EncodeFullFrame encodes the entire framebuffer without delta optimization.
// Useful for initial render or after terminal resize.
func (e *HalfBlockEncoder) EncodeFullFrame(fb *framebuffer.Framebuffer) []byte {
	e.buf.Reset()
	e.lastFG = color.RGBA{}
	e.lastBG = color.RGBA{}

	for row := 0; row < e.gridHeight; row++ {
		// Position cursor at start of row
		fmt.Fprintf(&e.buf, "\x1b[%d;1H", row+1)

		for col := 0; col < e.gridWidth; col++ {
			topY := row * 2
			botY := row*2 + 1

			var top, bot color.RGBA
			if topY < fb.Height && col < fb.Width {
				top = fb.Pixels.RGBAAt(col, topY)
			}
			if botY < fb.Height && col < fb.Width {
				bot = fb.Pixels.RGBAAt(col, botY)
			}

			// Update previous frame cache
			idx := row*e.gridWidth + col
			e.prevFrame[idx] = HalfBlockCell{Top: top, Bottom: bot}

			// Emit colors with RLE
			if top != e.lastFG {
				fmt.Fprintf(&e.buf, "\x1b[38;2;%d;%d;%dm", top.R, top.G, top.B)
				e.lastFG = top
			}
			if bot != e.lastBG {
				fmt.Fprintf(&e.buf, "\x1b[48;2;%d;%d;%dm", bot.R, bot.G, bot.B)
				e.lastBG = bot
			}

			e.buf.WriteString("▀")
		}
	}

	e.buf.WriteString("\x1b[0m")
	return e.buf.Bytes()
}

// cellsEqual checks if two cells are perceptually equal
func (e *HalfBlockEncoder) cellsEqual(a, b HalfBlockCell) bool {
	if e.threshold <= 0 {
		// Exact comparison
		return a.Top == b.Top && a.Bottom == b.Bottom
	}

	// Perceptual comparison
	topEqual := PerceptuallyEqual(
		a.Top.R, a.Top.G, a.Top.B,
		b.Top.R, b.Top.G, b.Top.B,
		e.threshold,
	)
	botEqual := PerceptuallyEqual(
		a.Bottom.R, a.Bottom.G, a.Bottom.B,
		b.Bottom.R, b.Bottom.G, b.Bottom.B,
		e.threshold,
	)
	return topEqual && botEqual
}

// EstimateBandwidth returns estimated bytes per frame for the given grid size.
// Useful for performance planning.
func EstimateBandwidth(gridWidth, gridHeight int, deltaPercent float64) int {
	totalCells := gridWidth * gridHeight
	changedCells := int(float64(totalCells) * deltaPercent)

	// Bytes per cell (worst case):
	// - Cursor position: ~10 bytes
	// - FG color: ~19 bytes
	// - BG color: ~19 bytes
	// - Character: 3 bytes (UTF-8 ▀)
	// Total: ~51 bytes
	// With color RLE, typically ~30 bytes average
	const bytesPerCell = 30

	return changedCells * bytesPerCell
}
