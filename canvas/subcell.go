// Package canvas provides the terminal framebuffer abstraction.
package canvas

// SubCellMode defines the sub-cell rendering mode.
type SubCellMode int

const (
	// SubCellNone uses standard single-character cells.
	SubCellNone SubCellMode = iota

	// SubCellHalfBlock uses Unicode half-block characters (▀▄█ )
	// to double vertical resolution (2 pixels per cell).
	SubCellHalfBlock

	// SubCellBraille uses Unicode Braille characters
	// for 2x4 pixels per cell (8 dots per character).
	SubCellBraille

	// SubCellQuadrant uses Unicode quadrant characters
	// for 2x2 pixels per cell (4 quadrants per character).
	SubCellQuadrant
)

// Half-block characters
const (
	HalfBlockUpper = '▀' // Upper half block
	HalfBlockLower = '▄' // Lower half block
	HalfBlockFull  = '█' // Full block
	HalfBlockEmpty = ' ' // Empty (space)
)

// Braille dot positions (Unicode Braille patterns U+2800 to U+28FF)
// Dots are numbered:
// 1 4
// 2 5
// 3 6
// 7 8
const (
	BrailleBase   = '\u2800'
	BrailleDot1   = 0x01
	BrailleDot2   = 0x02
	BrailleDot3   = 0x04
	BrailleDot4   = 0x08
	BrailleDot5   = 0x10
	BrailleDot6   = 0x20
	BrailleDot7   = 0x40
	BrailleDot8   = 0x80
	BrailleDotAll = 0xFF
)

// Quadrant block characters
const (
	QuadrantUpperLeft  = '▘'
	QuadrantUpperRight = '▝'
	QuadrantLowerLeft  = '▖'
	QuadrantLowerRight = '▗'
	QuadrantUpperBoth  = '▀'
	QuadrantLowerBoth  = '▄'
	QuadrantLeftBoth   = '▌'
	QuadrantRightBoth  = '▐'
	QuadrantDiagTLBR   = '▚' // Top-left and bottom-right
	QuadrantDiagTRBL   = '▞' // Top-right and bottom-left
	QuadrantMissingBR  = '▛' // All except bottom-right
	QuadrantMissingBL  = '▜' // All except bottom-left
	QuadrantMissingTR  = '▙' // All except top-right
	QuadrantMissingTL  = '▟' // All except top-left
	QuadrantFull       = '█'
	QuadrantEmpty      = ' '
)

// SubCellCanvas provides high-resolution rendering using Unicode characters.
type SubCellCanvas struct {
	canvas *Canvas
	mode   SubCellMode

	// Virtual pixel dimensions (higher resolution than cell dimensions)
	pixelWidth  int
	pixelHeight int

	// For half-block mode: store pixel values
	halfBlockPixels []bool // true = pixel on

	// For braille mode: store dot patterns
	braillePixels []bool // 2x4 grid per cell

	// For quadrant mode: store quadrant values
	quadrantPixels []bool // 2x2 grid per cell
}

// NewSubCellCanvas creates a new sub-cell canvas wrapping an existing canvas.
func NewSubCellCanvas(c *Canvas, mode SubCellMode) *SubCellCanvas {
	sc := &SubCellCanvas{
		canvas: c,
		mode:   mode,
	}
	sc.updateDimensions()
	return sc
}

// updateDimensions calculates pixel dimensions based on mode.
func (sc *SubCellCanvas) updateDimensions() {
	cellWidth := sc.canvas.Width()
	cellHeight := sc.canvas.Height()

	switch sc.mode {
	case SubCellHalfBlock:
		sc.pixelWidth = cellWidth
		sc.pixelHeight = cellHeight * 2
		sc.halfBlockPixels = make([]bool, sc.pixelWidth*sc.pixelHeight)

	case SubCellBraille:
		sc.pixelWidth = cellWidth * 2
		sc.pixelHeight = cellHeight * 4
		sc.braillePixels = make([]bool, sc.pixelWidth*sc.pixelHeight)

	case SubCellQuadrant:
		sc.pixelWidth = cellWidth * 2
		sc.pixelHeight = cellHeight * 2
		sc.quadrantPixels = make([]bool, sc.pixelWidth*sc.pixelHeight)

	default:
		sc.pixelWidth = cellWidth
		sc.pixelHeight = cellHeight
	}
}

// Width returns the pixel width.
func (sc *SubCellCanvas) Width() int {
	return sc.pixelWidth
}

// Height returns the pixel height.
func (sc *SubCellCanvas) Height() int {
	return sc.pixelHeight
}

// CellWidth returns the underlying cell width.
func (sc *SubCellCanvas) CellWidth() int {
	return sc.canvas.Width()
}

// CellHeight returns the underlying cell height.
func (sc *SubCellCanvas) CellHeight() int {
	return sc.canvas.Height()
}

// SetPixel sets a pixel at the given virtual coordinates.
func (sc *SubCellCanvas) SetPixel(x, y int, on bool) {
	if x < 0 || x >= sc.pixelWidth || y < 0 || y >= sc.pixelHeight {
		return
	}

	switch sc.mode {
	case SubCellHalfBlock:
		sc.halfBlockPixels[y*sc.pixelWidth+x] = on

	case SubCellBraille:
		sc.braillePixels[y*sc.pixelWidth+x] = on

	case SubCellQuadrant:
		sc.quadrantPixels[y*sc.pixelWidth+x] = on

	default:
		// Standard mode: directly set cell
		if on {
			sc.canvas.SetRune(x, y, HalfBlockFull)
		} else {
			sc.canvas.SetRune(x, y, ' ')
		}
	}
}

// GetPixel returns the pixel value at the given coordinates.
func (sc *SubCellCanvas) GetPixel(x, y int) bool {
	if x < 0 || x >= sc.pixelWidth || y < 0 || y >= sc.pixelHeight {
		return false
	}

	switch sc.mode {
	case SubCellHalfBlock:
		return sc.halfBlockPixels[y*sc.pixelWidth+x]
	case SubCellBraille:
		return sc.braillePixels[y*sc.pixelWidth+x]
	case SubCellQuadrant:
		return sc.quadrantPixels[y*sc.pixelWidth+x]
	default:
		cell := sc.canvas.GetCell(x, y)
		return cell.Rune != ' '
	}
}

// Clear clears all pixels.
func (sc *SubCellCanvas) Clear() {
	switch sc.mode {
	case SubCellHalfBlock:
		for i := range sc.halfBlockPixels {
			sc.halfBlockPixels[i] = false
		}
	case SubCellBraille:
		for i := range sc.braillePixels {
			sc.braillePixels[i] = false
		}
	case SubCellQuadrant:
		for i := range sc.quadrantPixels {
			sc.quadrantPixels[i] = false
		}
	}
	sc.canvas.Clear()
}

// Flush renders the sub-cell pixels to the underlying canvas.
func (sc *SubCellCanvas) Flush(fg, bg Color) {
	switch sc.mode {
	case SubCellHalfBlock:
		sc.flushHalfBlock(fg, bg)
	case SubCellBraille:
		sc.flushBraille(fg)
	case SubCellQuadrant:
		sc.flushQuadrant(fg, bg)
	}
}

// flushHalfBlock renders half-block pixels to the canvas.
func (sc *SubCellCanvas) flushHalfBlock(fg, bg Color) {
	cellWidth := sc.canvas.Width()
	cellHeight := sc.canvas.Height()

	for cy := 0; cy < cellHeight; cy++ {
		for cx := 0; cx < cellWidth; cx++ {
			topY := cy * 2
			botY := cy*2 + 1

			top := sc.halfBlockPixels[topY*sc.pixelWidth+cx]
			bot := sc.halfBlockPixels[botY*sc.pixelWidth+cx]

			var r rune
			var cell Cell

			if top && bot {
				r = HalfBlockFull
				cell = NewCell(r, fg)
			} else if top && !bot {
				r = HalfBlockUpper
				cell = NewCell(r, fg)
			} else if !top && bot {
				r = HalfBlockLower
				cell = NewCell(r, fg)
			} else {
				r = ' '
				cell = NewCell(r, bg)
			}

			sc.canvas.SetCell(cx, cy, cell)
		}
	}
}

// flushBraille renders braille pixels to the canvas.
func (sc *SubCellCanvas) flushBraille(fg Color) {
	cellWidth := sc.canvas.Width()
	cellHeight := sc.canvas.Height()

	for cy := 0; cy < cellHeight; cy++ {
		for cx := 0; cx < cellWidth; cx++ {
			pattern := rune(0)

			// Map pixel positions to braille dots:
			// Pixel (0,0) -> Dot 1, (1,0) -> Dot 4
			// Pixel (0,1) -> Dot 2, (1,1) -> Dot 5
			// Pixel (0,2) -> Dot 3, (1,2) -> Dot 6
			// Pixel (0,3) -> Dot 7, (1,3) -> Dot 8

			px := cx * 2
			py := cy * 4

			if sc.getBraillePixel(px, py) {
				pattern |= BrailleDot1
			}
			if sc.getBraillePixel(px+1, py) {
				pattern |= BrailleDot4
			}
			if sc.getBraillePixel(px, py+1) {
				pattern |= BrailleDot2
			}
			if sc.getBraillePixel(px+1, py+1) {
				pattern |= BrailleDot5
			}
			if sc.getBraillePixel(px, py+2) {
				pattern |= BrailleDot3
			}
			if sc.getBraillePixel(px+1, py+2) {
				pattern |= BrailleDot6
			}
			if sc.getBraillePixel(px, py+3) {
				pattern |= BrailleDot7
			}
			if sc.getBraillePixel(px+1, py+3) {
				pattern |= BrailleDot8
			}

			sc.canvas.SetCell(cx, cy, NewCell(BrailleBase+pattern, fg))
		}
	}
}

func (sc *SubCellCanvas) getBraillePixel(x, y int) bool {
	if x < 0 || x >= sc.pixelWidth || y < 0 || y >= sc.pixelHeight {
		return false
	}
	return sc.braillePixels[y*sc.pixelWidth+x]
}

// flushQuadrant renders quadrant pixels to the canvas.
func (sc *SubCellCanvas) flushQuadrant(fg, bg Color) {
	cellWidth := sc.canvas.Width()
	cellHeight := sc.canvas.Height()

	for cy := 0; cy < cellHeight; cy++ {
		for cx := 0; cx < cellWidth; cx++ {
			px := cx * 2
			py := cy * 2

			tl := sc.getQuadrantPixel(px, py)
			tr := sc.getQuadrantPixel(px+1, py)
			bl := sc.getQuadrantPixel(px, py+1)
			br := sc.getQuadrantPixel(px+1, py+1)

			r := quadrantChar(tl, tr, bl, br)
			sc.canvas.SetCell(cx, cy, NewCell(r, fg))
		}
	}
}

func (sc *SubCellCanvas) getQuadrantPixel(x, y int) bool {
	if x < 0 || x >= sc.pixelWidth || y < 0 || y >= sc.pixelHeight {
		return false
	}
	return sc.quadrantPixels[y*sc.pixelWidth+x]
}

// quadrantChar returns the appropriate quadrant character for the given pattern.
func quadrantChar(tl, tr, bl, br bool) rune {
	pattern := 0
	if tl {
		pattern |= 1
	}
	if tr {
		pattern |= 2
	}
	if bl {
		pattern |= 4
	}
	if br {
		pattern |= 8
	}

	// Lookup table for all 16 combinations
	chars := []rune{
		' ',               // 0000
		QuadrantUpperLeft, // 0001
		QuadrantUpperRight,// 0010
		QuadrantUpperBoth, // 0011
		QuadrantLowerLeft, // 0100
		QuadrantLeftBoth,  // 0101
		QuadrantDiagTRBL,  // 0110
		QuadrantMissingBR, // 0111
		QuadrantLowerRight,// 1000
		QuadrantDiagTLBR,  // 1001
		QuadrantRightBoth, // 1010
		QuadrantMissingBL, // 1011
		QuadrantLowerBoth, // 1100
		QuadrantMissingTR, // 1101
		QuadrantMissingTL, // 1110
		QuadrantFull,      // 1111
	}

	return chars[pattern]
}

// Canvas returns the underlying canvas.
func (sc *SubCellCanvas) Canvas() *Canvas {
	return sc.canvas
}

// String renders the canvas to a string via the underlying canvas.
func (sc *SubCellCanvas) String() string {
	return sc.canvas.String()
}

// DrawLine draws a line in pixel coordinates.
func (sc *SubCellCanvas) DrawLine(x1, y1, x2, y2 int) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		sc.SetPixel(x1, y1, true)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// DrawRect draws a rectangle outline in pixel coordinates.
func (sc *SubCellCanvas) DrawRect(x1, y1, x2, y2 int) {
	sc.DrawLine(x1, y1, x2, y1)
	sc.DrawLine(x2, y1, x2, y2)
	sc.DrawLine(x2, y2, x1, y2)
	sc.DrawLine(x1, y2, x1, y1)
}

// FillRect fills a rectangle in pixel coordinates.
func (sc *SubCellCanvas) FillRect(x1, y1, x2, y2 int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			sc.SetPixel(x, y, true)
		}
	}
}

// DrawCircle draws a circle outline using Bresenham's algorithm.
func (sc *SubCellCanvas) DrawCircle(cx, cy, r int) {
	x := 0
	y := r
	d := 3 - 2*r

	sc.setCirclePoints(cx, cy, x, y)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		sc.setCirclePoints(cx, cy, x, y)
	}
}

func (sc *SubCellCanvas) setCirclePoints(cx, cy, x, y int) {
	sc.SetPixel(cx+x, cy+y, true)
	sc.SetPixel(cx-x, cy+y, true)
	sc.SetPixel(cx+x, cy-y, true)
	sc.SetPixel(cx-x, cy-y, true)
	sc.SetPixel(cx+y, cy+x, true)
	sc.SetPixel(cx-y, cy+x, true)
	sc.SetPixel(cx+y, cy-x, true)
	sc.SetPixel(cx-y, cy-x, true)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// BrailleChar creates a Braille character from an 8-bit pattern.
// Bits correspond to dots: 1=dot1, 2=dot2, 4=dot3, 8=dot4, 16=dot5, 32=dot6, 64=dot7, 128=dot8
func BrailleChar(pattern byte) rune {
	return BrailleBase + rune(pattern)
}

// BrailleCharFromDots creates a Braille character from individual dot states.
func BrailleCharFromDots(d1, d2, d3, d4, d5, d6, d7, d8 bool) rune {
	var pattern byte
	if d1 {
		pattern |= BrailleDot1
	}
	if d2 {
		pattern |= BrailleDot2
	}
	if d3 {
		pattern |= BrailleDot3
	}
	if d4 {
		pattern |= BrailleDot4
	}
	if d5 {
		pattern |= BrailleDot5
	}
	if d6 {
		pattern |= BrailleDot6
	}
	if d7 {
		pattern |= BrailleDot7
	}
	if d8 {
		pattern |= BrailleDot8
	}
	return BrailleChar(pattern)
}
