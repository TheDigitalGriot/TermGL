package tier2

import (
	"image"
	"image/color"

	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
)

// ANSIOutput implements render.Encoder (and render.AuxEncoder) for Tier 2.
// Architecture doc Section 6.1
type ANSIOutput struct {
	blitter   Blitter
	splitter  *FrequencySplitter
	selector  *EdgeAwareSelector
	delta     *DeltaEncoder
	dither    tier1.DitherMode

	// Options
	useFrequencySplit bool
	useEdgeAware      bool
	useDelta          bool

	termCols int
	termRows int
}

// NewANSIOutput creates a new ANSI subpixel encoder.
func NewANSIOutput(blitter Blitter) *ANSIOutput {
	return &ANSIOutput{
		blitter:           blitter,
		splitter:          &FrequencySplitter{},
		dither:            tier1.DitherOrdered4x4,
		useFrequencySplit: true,
		useEdgeAware:      true,
		useDelta:          true,
	}
}

// WithDither sets the dithering mode.
func (a *ANSIOutput) WithDither(mode tier1.DitherMode) *ANSIOutput {
	a.dither = mode
	return a
}

// WithFrequencySplit enables or disables frequency splitting.
func (a *ANSIOutput) WithFrequencySplit(enabled bool) *ANSIOutput {
	a.useFrequencySplit = enabled
	return a
}

// WithEdgeAware enables or disables edge-aware character selection.
func (a *ANSIOutput) WithEdgeAware(enabled bool) *ANSIOutput {
	a.useEdgeAware = enabled
	return a
}

// WithDeltaEncoding enables or disables delta encoding.
func (a *ANSIOutput) WithDeltaEncoding(enabled bool) *ANSIOutput {
	a.useDelta = enabled
	return a
}

// Init sets up the encoder for the given terminal capabilities.
// Implements render.Encoder.
func (a *ANSIOutput) Init(caps render.TerminalCaps) error {
	a.termCols = caps.Width
	a.termRows = caps.Height

	// Create delta encoder
	if a.useDelta {
		a.delta = NewDeltaEncoder(a.termCols, a.termRows)
	}

	return nil
}

// Encode converts a rendered frame to ANSI escape sequences.
// Returns the string to write to stdout.
func (a *ANSIOutput) Encode(frame *image.NRGBA) string {
	cells := a.encodeFrame(frame, nil)

	if a.useDelta && a.delta != nil {
		return a.delta.Encode(cells)
	}

	// No delta encoding - emit all cells
	return encodeCellsFull(cells, a.termCols)
}

// EncodeWithCursor converts a rendered frame with cursor positioning.
func (a *ANSIOutput) EncodeWithCursor(frame *image.NRGBA, row, col int) string {
	cells := a.encodeFrame(frame, nil)

	if a.useDelta && a.delta != nil {
		return a.delta.EncodeWithCursor(cells, row, col)
	}

	// No delta encoding
	return encodeCellsFullWithCursor(cells, a.termCols, row, col)
}

// EncodeWithAux converts a rendered frame using auxiliary buffers.
func (a *ANSIOutput) EncodeWithAux(frame *image.NRGBA, aux *render.AuxBuffers) string {
	cells := a.encodeFrame(frame, aux)

	if a.useDelta && a.delta != nil {
		return a.delta.Encode(cells)
	}

	return encodeCellsFull(cells, a.termCols)
}

// EncodeWithAuxAndCursor converts a rendered frame using auxiliary buffers with cursor positioning.
func (a *ANSIOutput) EncodeWithAuxAndCursor(frame *image.NRGBA, aux *render.AuxBuffers, row, col int) string {
	cells := a.encodeFrame(frame, aux)

	if a.useDelta && a.delta != nil {
		return a.delta.EncodeWithCursor(cells, row, col)
	}

	return encodeCellsFullWithCursor(cells, a.termCols, row, col)
}

// encodeFrame is the core encoding pipeline.
func (a *ANSIOutput) encodeFrame(frame *image.NRGBA, aux *render.AuxBuffers) []Cell {
	bounds := frame.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	cellW, cellH := a.blitter.SubCellSize()

	// Calculate terminal dimensions from image size
	termCols := imgW / cellW
	termRows := imgH / cellH

	cells := make([]Cell, termCols*termRows)

	// Step 1: Frequency splitting (optional)
	if a.useFrequencySplit {
		a.splitter.Split(frame, cellW, cellH)

		// Step 2: Apply dithering to luminance channel
		if a.dither != tier1.DitherNone {
			BayerDither(a.splitter.Y, imgW, imgH, a.dither)
		}
	}

	// Step 3: Create edge-aware selector if we have aux buffers
	if a.useEdgeAware && aux != nil {
		a.selector = NewEdgeAwareSelector(a.blitter, aux.DepthMap, aux.NormalMap, imgW, imgH)
	}

	// Step 4: Encode each cell
	for ty := 0; ty < termRows; ty++ {
		for tx := 0; tx < termCols; tx++ {
			cellIdx := ty*termCols + tx

			// Extract luminance pattern for this cell
			lumaPattern := make([]float64, cellW*cellH)
			if a.useFrequencySplit {
				for sy := 0; sy < cellH; sy++ {
					for sx := 0; sx < cellW; sx++ {
						px := tx*cellW + sx
						py := ty*cellH + sy
						subIdx := py*imgW + px
						if subIdx < len(a.splitter.Y) {
							lumaPattern[sy*cellW+sx] = a.splitter.Y[subIdx]
						}
					}
				}
			} else {
				// Fallback: compute luminance directly from pixels
				for sy := 0; sy < cellH; sy++ {
					for sx := 0; sx < cellW; sx++ {
						px := bounds.Min.X + tx*cellW + sx
						py := bounds.Min.Y + ty*cellH + sy
						if px < bounds.Max.X && py < bounds.Max.Y {
							c := frame.NRGBAAt(px, py)
							luma := luminanceFromRGB(c)
							lumaPattern[sy*cellW+sx] = luma
						}
					}
				}
			}

			// Select character and colors
			var char rune
			var fg, bg color.NRGBA

			if a.useEdgeAware && a.selector != nil {
				char, fg, bg = a.selector.SelectChar(frame, lumaPattern, tx, ty, cellW, cellH)
			} else {
				// Simple threshold-based selection
				char, fg, bg = selectCharSimple(frame, lumaPattern, a.blitter, tx, ty, cellW, cellH)
			}

			// Apply chrominance to fg/bg colors if using frequency split
			if a.useFrequencySplit && cellIdx < len(a.splitter.Cb) {
				fg = applyChroma(fg, a.splitter.Cb[cellIdx], a.splitter.Cr[cellIdx])
				bg = applyChroma(bg, a.splitter.Cb[cellIdx], a.splitter.Cr[cellIdx])
			}

			cells[cellIdx] = Cell{
				Char: char,
				FG:   fg,
				BG:   bg,
			}
		}
	}

	return cells
}

// InternalResolution returns the pixel resolution the rasterizer should render at.
// ANSI renders at sub-cell resolution.
func (a *ANSIOutput) InternalResolution(termCols, termRows int) (int, int) {
	cellW, cellH := a.blitter.SubCellSize()
	return termCols * cellW, termRows * cellH
}

// selectCharSimple performs simple threshold-based character selection without edge awareness.
func selectCharSimple(
	img *image.NRGBA,
	lumaPattern []float64,
	blitter Blitter,
	cellX, cellY, cellW, cellH int,
) (rune, color.NRGBA, color.NRGBA) {

	// Threshold luminance values to binary pattern
	threshold := medianLuma(lumaPattern)
	pattern := uint8(0)
	numBits := cellW * cellH

	for i := 0; i < numBits && i < len(lumaPattern); i++ {
		if lumaPattern[i] >= threshold {
			pattern |= 1 << uint(i)
		}
	}

	// Look up character from blitter
	char := blitter.CharForPattern(pattern)

	// Collect pixel colors for this cell
	pixels := make([]color.NRGBA, 0, numBits)
	bounds := img.Bounds()
	for sy := 0; sy < cellH; sy++ {
		for sx := 0; sx < cellW; sx++ {
			px := bounds.Min.X + cellX*cellW + sx
			py := bounds.Min.Y + cellY*cellH + sy
			if px < bounds.Max.X && py < bounds.Max.Y {
				pixels = append(pixels, img.NRGBAAt(px, py))
			}
		}
	}

	// Compute fg/bg colors
	fg, bg := OptimizeCellColors(pixels, pattern, numBits)

	return char, fg, bg
}

// luminanceFromRGB computes luminance using BT.601 coefficients.
func luminanceFromRGB(c color.NRGBA) float64 {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0
	return 0.299*r + 0.587*g + 0.114*b
}

// applyChroma applies chrominance to a base color.
func applyChroma(base color.NRGBA, cb, cr float64) color.NRGBA {
	// Convert base to float
	r := float64(base.R) / 255.0
	g := float64(base.G) / 255.0
	b := float64(base.B) / 255.0

	// Compute luminance
	y := 0.299*r + 0.587*g + 0.114*b

	// Apply chrominance offsets
	r = y + cr
	b = y + cb
	g = (y - 0.299*r - 0.114*b) / 0.587

	// Clamp to [0, 1]
	r = clampF64(r, 0, 1)
	g = clampF64(g, 0, 1)
	b = clampF64(b, 0, 1)

	return color.NRGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

func clampF64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// encodeCellsFull encodes all cells without delta encoding.
func encodeCellsFull(cells []Cell, termCols int) string {
	var buf []byte

	lastFG := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	lastBG := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	for i, cell := range cells {
		// Emit newline at end of each row
		if i > 0 && i%termCols == 0 {
			buf = append(buf, '\n')
		}

		// Emit fg color if changed
		if cell.FG != lastFG {
			buf = append(buf, []byte("\x1b[38;2;")...)
			buf = appendInt(buf, int(cell.FG.R))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.FG.G))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.FG.B))
			buf = append(buf, 'm')
			lastFG = cell.FG
		}

		// Emit bg color if changed
		if cell.BG != lastBG {
			buf = append(buf, []byte("\x1b[48;2;")...)
			buf = appendInt(buf, int(cell.BG.R))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.BG.G))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.BG.B))
			buf = append(buf, 'm')
			lastBG = cell.BG
		}

		// Emit character
		buf = appendRune(buf, cell.Char)
	}

	// Reset colors
	buf = append(buf, []byte("\x1b[0m")...)

	return string(buf)
}

// encodeCellsFullWithCursor encodes all cells with cursor positioning.
func encodeCellsFullWithCursor(cells []Cell, termCols, row, col int) string {
	var buf []byte

	// Move to starting position
	buf = append(buf, []byte("\x1b[")...)
	buf = appendInt(buf, row)
	buf = append(buf, ';')
	buf = appendInt(buf, col)
	buf = append(buf, 'H')

	lastFG := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	lastBG := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	for i, cell := range cells {
		// Move cursor at start of each row (except first)
		if i > 0 && i%termCols == 0 {
			y := row + i/termCols
			buf = append(buf, []byte("\x1b[")...)
			buf = appendInt(buf, y)
			buf = append(buf, ';')
			buf = appendInt(buf, col)
			buf = append(buf, 'H')
		}

		// Emit fg color if changed
		if cell.FG != lastFG {
			buf = append(buf, []byte("\x1b[38;2;")...)
			buf = appendInt(buf, int(cell.FG.R))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.FG.G))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.FG.B))
			buf = append(buf, 'm')
			lastFG = cell.FG
		}

		// Emit bg color if changed
		if cell.BG != lastBG {
			buf = append(buf, []byte("\x1b[48;2;")...)
			buf = appendInt(buf, int(cell.BG.R))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.BG.G))
			buf = append(buf, ';')
			buf = appendInt(buf, int(cell.BG.B))
			buf = append(buf, 'm')
			lastBG = cell.BG
		}

		// Emit character
		buf = appendRune(buf, cell.Char)
	}

	// Reset colors
	buf = append(buf, []byte("\x1b[0m")...)

	return string(buf)
}

// appendInt appends an integer to a byte slice (fast, no allocations).
func appendInt(buf []byte, n int) []byte {
	if n == 0 {
		return append(buf, '0')
	}

	// Handle negative
	if n < 0 {
		buf = append(buf, '-')
		n = -n
	}

	// Count digits
	temp := n
	digits := 0
	for temp > 0 {
		digits++
		temp /= 10
	}

	// Reserve space
	start := len(buf)
	for i := 0; i < digits; i++ {
		buf = append(buf, '0')
	}

	// Write digits in reverse
	for i := digits - 1; i >= 0; i-- {
		buf[start+i] = byte('0' + n%10)
		n /= 10
	}

	return buf
}

// appendRune appends a rune to a byte slice.
func appendRune(buf []byte, r rune) []byte {
	if r < 128 {
		return append(buf, byte(r))
	}

	// UTF-8 encode
	var tmp [4]byte
	n := encodeRune(tmp[:], r)
	return append(buf, tmp[:n]...)
}

// encodeRune encodes a rune to UTF-8 (simplified).
func encodeRune(buf []byte, r rune) int {
	// Simple UTF-8 encoding
	if r < 0x80 {
		buf[0] = byte(r)
		return 1
	}
	if r < 0x800 {
		buf[0] = byte(0xC0 | (r >> 6))
		buf[1] = byte(0x80 | (r & 0x3F))
		return 2
	}
	if r < 0x10000 {
		buf[0] = byte(0xE0 | (r >> 12))
		buf[1] = byte(0x80 | ((r >> 6) & 0x3F))
		buf[2] = byte(0x80 | (r & 0x3F))
		return 3
	}
	buf[0] = byte(0xF0 | (r >> 18))
	buf[1] = byte(0x80 | ((r >> 12) & 0x3F))
	buf[2] = byte(0x80 | ((r >> 6) & 0x3F))
	buf[3] = byte(0x80 | (r & 0x3F))
	return 4
}
