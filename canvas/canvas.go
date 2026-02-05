package canvas

import (
	"math"
)

// Canvas is a 2D grid of terminal cells serving as the framebuffer.
type Canvas struct {
	width   int
	height  int
	cells   []Cell
	zBuffer []float64
}

// New creates a new canvas with the given dimensions.
func New(width, height int) *Canvas {
	size := width * height
	cells := make([]Cell, size)
	zBuffer := make([]float64, size)

	// Initialize cells and z-buffer
	for i := range cells {
		cells[i] = DefaultCell()
		zBuffer[i] = math.MaxFloat64
	}

	return &Canvas{
		width:   width,
		height:  height,
		cells:   cells,
		zBuffer: zBuffer,
	}
}

// Width returns the canvas width in cells.
func (c *Canvas) Width() int {
	return c.width
}

// Height returns the canvas height in cells.
func (c *Canvas) Height() int {
	return c.height
}

// InBounds returns true if (x, y) is within the canvas bounds.
func (c *Canvas) InBounds(x, y int) bool {
	return x >= 0 && x < c.width && y >= 0 && y < c.height
}

// index converts (x, y) to array index.
func (c *Canvas) index(x, y int) int {
	return y*c.width + x
}

// SetCell sets a cell at the given position.
func (c *Canvas) SetCell(x, y int, cell Cell) {
	if !c.InBounds(x, y) {
		return
	}
	c.cells[c.index(x, y)] = cell
}

// GetCell returns the cell at the given position.
func (c *Canvas) GetCell(x, y int) Cell {
	if !c.InBounds(x, y) {
		return DefaultCell()
	}
	return c.cells[c.index(x, y)]
}

// SetRune sets just the rune at a position, preserving colors.
func (c *Canvas) SetRune(x, y int, r rune) {
	if !c.InBounds(x, y) {
		return
	}
	idx := c.index(x, y)
	c.cells[idx].Rune = r
}

// SetForeground sets just the foreground color at a position.
func (c *Canvas) SetForeground(x, y int, fg Color) {
	if !c.InBounds(x, y) {
		return
	}
	idx := c.index(x, y)
	c.cells[idx].Foreground = fg
}

// SetBackground sets just the background color at a position.
func (c *Canvas) SetBackground(x, y int, bg Color) {
	if !c.InBounds(x, y) {
		return
	}
	idx := c.index(x, y)
	c.cells[idx].Background = bg
}

// Clear resets all cells to empty and clears the z-buffer.
func (c *Canvas) Clear() {
	for i := range c.cells {
		c.cells[i] = DefaultCell()
		c.zBuffer[i] = math.MaxFloat64
	}
}

// ClearDepth resets only the z-buffer to max depth.
func (c *Canvas) ClearDepth() {
	for i := range c.zBuffer {
		c.zBuffer[i] = math.MaxFloat64
	}
}

// Resize changes the canvas dimensions.
func (c *Canvas) Resize(width, height int) {
	if width == c.width && height == c.height {
		return
	}

	size := width * height
	newCells := make([]Cell, size)
	newZBuffer := make([]float64, size)

	// Initialize new cells
	for i := range newCells {
		newCells[i] = DefaultCell()
		newZBuffer[i] = math.MaxFloat64
	}

	// Copy existing content where it fits
	minWidth := c.width
	if width < minWidth {
		minWidth = width
	}
	minHeight := c.height
	if height < minHeight {
		minHeight = height
	}

	for y := 0; y < minHeight; y++ {
		for x := 0; x < minWidth; x++ {
			oldIdx := y*c.width + x
			newIdx := y*width + x
			newCells[newIdx] = c.cells[oldIdx]
			newZBuffer[newIdx] = c.zBuffer[oldIdx]
		}
	}

	c.width = width
	c.height = height
	c.cells = newCells
	c.zBuffer = newZBuffer
}

// Z-buffer methods

// SetDepth sets the z-buffer depth at a position.
func (c *Canvas) SetDepth(x, y int, depth float64) {
	if !c.InBounds(x, y) {
		return
	}
	c.zBuffer[c.index(x, y)] = depth
}

// GetDepth returns the z-buffer depth at a position.
func (c *Canvas) GetDepth(x, y int) float64 {
	if !c.InBounds(x, y) {
		return math.MaxFloat64
	}
	return c.zBuffer[c.index(x, y)]
}

// TestAndSetDepth atomically tests if depth is less than current z-buffer value.
// If so, it updates the z-buffer and returns true. Otherwise returns false.
func (c *Canvas) TestAndSetDepth(x, y int, depth float64) bool {
	if !c.InBounds(x, y) {
		return false
	}
	idx := c.index(x, y)
	if depth < c.zBuffer[idx] {
		c.zBuffer[idx] = depth
		return true
	}
	return false
}

// Aspect returns the aspect ratio (width / height).
func (c *Canvas) Aspect() float64 {
	if c.height == 0 {
		return 1.0
	}
	return float64(c.width) / float64(c.height)
}
