package tier2

import (
	"bytes"
	"fmt"
	"image/color"
)

// Cell represents one terminal cell's encoded output.
// Architecture doc Section 5.6
type Cell struct {
	Char rune
	FG   color.NRGBA
	BG   color.NRGBA
}

// DeltaEncoder tracks the previous frame and only emits changes.
// Architecture doc Section 5.6
type DeltaEncoder struct {
	prevCells []Cell
	termCols  int
	termRows  int
	threshold float64 // perceptual threshold (default: 2.0 JND)
}

// NewDeltaEncoder creates a new delta encoder.
func NewDeltaEncoder(termCols, termRows int) *DeltaEncoder {
	return &DeltaEncoder{
		prevCells: make([]Cell, termCols*termRows),
		termCols:  termCols,
		termRows:  termRows,
		threshold: 2.0, // Just noticeable difference in perceptual color space
	}
}

// Encode writes only the changed cells as ANSI escape sequences.
func (d *DeltaEncoder) Encode(cells []Cell) string {
	var buf bytes.Buffer

	lastFG := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	lastBG := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	lastX, lastY := -1, -1

	for i, cell := range cells {
		x := i % d.termCols
		y := i / d.termCols

		// Skip if cell hasn't changed (within perceptual threshold)
		if i < len(d.prevCells) && d.cellUnchanged(d.prevCells[i], cell) {
			continue
		}

		// Emit cursor movement if not sequential
		if y != lastY || x != lastX+1 {
			fmt.Fprintf(&buf, "\x1b[%d;%dH", y+1, x+1)
		}

		// Emit fg color if changed
		if cell.FG != lastFG {
			fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm", cell.FG.R, cell.FG.G, cell.FG.B)
			lastFG = cell.FG
		}

		// Emit bg color if changed
		if cell.BG != lastBG {
			fmt.Fprintf(&buf, "\x1b[48;2;%d;%d;%dm", cell.BG.R, cell.BG.G, cell.BG.B)
			lastBG = cell.BG
		}

		// Emit character
		buf.WriteRune(cell.Char)

		lastX, lastY = x, y
	}

	// Store for next frame's delta comparison
	if len(d.prevCells) == len(cells) {
		copy(d.prevCells, cells)
	} else {
		d.prevCells = make([]Cell, len(cells))
		copy(d.prevCells, cells)
	}

	return buf.String()
}

// EncodeWithCursor encodes with explicit cursor positioning at the start.
func (d *DeltaEncoder) EncodeWithCursor(cells []Cell, row, col int) string {
	var buf bytes.Buffer

	// Move to starting position
	fmt.Fprintf(&buf, "\x1b[%d;%dH", row, col)

	lastFG := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	lastBG := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	lastX, lastY := -1, -1

	for i, cell := range cells {
		x := i % d.termCols
		y := i / d.termCols

		// Skip if cell hasn't changed (within perceptual threshold)
		if i < len(d.prevCells) && d.cellUnchanged(d.prevCells[i], cell) {
			continue
		}

		// Emit cursor movement if not sequential
		if y != lastY || x != lastX+1 {
			fmt.Fprintf(&buf, "\x1b[%d;%dH", row+y, col+x)
		}

		// Emit fg color if changed
		if cell.FG != lastFG {
			fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm", cell.FG.R, cell.FG.G, cell.FG.B)
			lastFG = cell.FG
		}

		// Emit bg color if changed
		if cell.BG != lastBG {
			fmt.Fprintf(&buf, "\x1b[48;2;%d;%d;%dm", cell.BG.R, cell.BG.G, cell.BG.B)
			lastBG = cell.BG
		}

		// Emit character
		buf.WriteRune(cell.Char)

		lastX, lastY = x, y
	}

	// Store for next frame's delta comparison
	if len(d.prevCells) == len(cells) {
		copy(d.prevCells, cells)
	} else {
		d.prevCells = make([]Cell, len(cells))
		copy(d.prevCells, cells)
	}

	return buf.String()
}

// Reset clears delta state (forces full redraw next frame).
func (d *DeltaEncoder) Reset() {
	for i := range d.prevCells {
		d.prevCells[i] = Cell{}
	}
}

// cellUnchanged checks if a cell has changed significantly.
func (d *DeltaEncoder) cellUnchanged(prev, curr Cell) bool {
	// Character must match
	if prev.Char != curr.Char {
		return false
	}

	// Colors must be within perceptual threshold
	// For performance, use simple RGB distance rather than full DIN99d
	fgDist := colorDistanceRGB(prev.FG, curr.FG)
	bgDist := colorDistanceRGB(prev.BG, curr.BG)

	// Threshold in RGB space (approximate)
	const rgbThreshold = 8.0 // roughly 2 JND

	return fgDist < rgbThreshold && bgDist < rgbThreshold
}

// colorDistanceRGB computes simple Euclidean distance in RGB space.
func colorDistanceRGB(c1, c2 color.NRGBA) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return dr*dr + dg*dg + db*db
}
