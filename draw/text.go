// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"github.com/charmbracelet/termgl/canvas"
)

// PutString draws a string at the specified position.
// Each character is drawn in its own cell, advancing x by 1 for each character.
func PutString(c *canvas.Canvas, x, y int, s string, cell canvas.Cell) {
	for i, r := range s {
		cell.Rune = r
		c.SetCell(x+i, y, cell)
	}
}

// PutStringColor draws a string at the specified position with the given foreground color.
func PutStringColor(c *canvas.Canvas, x, y int, s string, fg canvas.Color) {
	for i, r := range s {
		c.SetCell(x+i, y, canvas.NewCell(r, fg))
	}
}

// PutStringStyled draws a string with per-character styling.
// The styler function receives the index and rune, and returns the cell to draw.
type CellStyler func(index int, r rune) canvas.Cell

func PutStringStyled(c *canvas.Canvas, x, y int, s string, styler CellStyler) {
	for i, r := range s {
		c.SetCell(x+i, y, styler(i, r))
	}
}

// PutChar draws a single character at the specified position.
// This is an alias for canvas.SetCell but with a more intuitive API.
func PutChar(c *canvas.Canvas, x, y int, r rune, fg canvas.Color) {
	c.SetCell(x, y, canvas.NewCell(r, fg))
}

// PutCharCell draws a single character cell at the specified position.
func PutCharCell(c *canvas.Canvas, x, y int, cell canvas.Cell) {
	c.SetCell(x, y, cell)
}

// MeasureString returns the width of a string in cells.
// For ASCII strings, this equals len(s).
// For unicode strings, this counts the number of runes.
func MeasureString(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// PutStringCentered draws a string centered horizontally on the canvas at the given y.
func PutStringCentered(c *canvas.Canvas, y int, s string, cell canvas.Cell) {
	width := c.Width()
	strLen := MeasureString(s)
	x := (width - strLen) / 2
	PutString(c, x, y, s, cell)
}

// PutStringRight draws a string right-aligned at the given x coordinate.
// The string ends at x (x is the rightmost position).
func PutStringRight(c *canvas.Canvas, x, y int, s string, cell canvas.Cell) {
	strLen := MeasureString(s)
	startX := x - strLen + 1
	PutString(c, startX, y, s, cell)
}

// PutStringVertical draws a string vertically, with each character on a new line.
func PutStringVertical(c *canvas.Canvas, x, y int, s string, cell canvas.Cell) {
	for i, r := range s {
		cell.Rune = r
		c.SetCell(x, y+i, cell)
	}
}

// PutStringWrap draws a string with word wrapping at the specified width.
// Returns the number of lines used.
func PutStringWrap(c *canvas.Canvas, x, y, maxWidth int, s string, cell canvas.Cell) int {
	if maxWidth <= 0 {
		return 0
	}

	line := 0
	col := 0
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Handle newline
		if r == '\n' {
			line++
			col = 0
			continue
		}

		// Wrap if we've exceeded the width
		if col >= maxWidth {
			line++
			col = 0
		}

		cell.Rune = r
		c.SetCell(x+col, y+line, cell)
		col++
	}

	return line + 1
}

// ClearLine clears a horizontal line by filling it with spaces.
func ClearLine(c *canvas.Canvas, y int) {
	width := c.Width()
	cell := canvas.DefaultCell()
	for x := 0; x < width; x++ {
		c.SetCell(x, y, cell)
	}
}

// ClearRect clears a rectangular region by filling it with spaces.
func ClearRect(c *canvas.Canvas, x1, y1, x2, y2 int) {
	cell := canvas.DefaultCell()
	FillRect(c, x1, y1, x2, y2, cell)
}
