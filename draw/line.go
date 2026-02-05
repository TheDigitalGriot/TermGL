// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"github.com/charmbracelet/termgl/canvas"
	"math"
)

// Line draws a line between two points using Bresenham-style algorithm.
// This is ported from ascii-graphics-3d/src/Screen.cpp:124-158
func Line(c *canvas.Canvas, x1, y1, x2, y2 int, cell canvas.Cell) {
	// Handle vertical line or single point
	if x1 == x2 {
		if y1 == y2 {
			c.SetCell(x1, y1, cell)
			return
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			c.SetCell(x1, y, cell)
		}
		return
	}

	// General case - use slope-intercept form
	slope := float64(y2-y1) / float64(x2-x1)
	intercept := float64(y1) - slope*float64(x1)

	// Draw along X axis
	startX, endX := x1, x2
	if startX > endX {
		startX, endX = endX, startX
	}
	for x := startX; x <= endX; x++ {
		y := int(math.Round(slope*float64(x) + intercept))
		c.SetCell(x, y, cell)
	}

	// Draw along Y axis to fill gaps in steep lines
	startY, endY := y1, y2
	if startY > endY {
		startY, endY = endY, startY
	}
	for y := startY; y <= endY; y++ {
		x := int(math.Round((float64(y) - intercept) / slope))
		c.SetCell(x, y, cell)
	}
}

// LineFloat draws a line between two float coordinates.
func LineFloat(c *canvas.Canvas, x1, y1, x2, y2 float64, cell canvas.Cell) {
	Line(c, int(math.Round(x1)), int(math.Round(y1)), int(math.Round(x2)), int(math.Round(y2)), cell)
}

// HLine draws a horizontal line (optimized).
func HLine(c *canvas.Canvas, x1, x2, y int, cell canvas.Cell) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		c.SetCell(x, y, cell)
	}
}

// VLine draws a vertical line (optimized).
func VLine(c *canvas.Canvas, x, y1, y2 int, cell canvas.Cell) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		c.SetCell(x, y, cell)
	}
}

// Rect draws a rectangle outline.
func Rect(c *canvas.Canvas, x1, y1, x2, y2 int, cell canvas.Cell) {
	HLine(c, x1, x2, y1, cell)
	HLine(c, x1, x2, y2, cell)
	VLine(c, x1, y1, y2, cell)
	VLine(c, x2, y1, y2, cell)
}

// FillRect fills a rectangle.
func FillRect(c *canvas.Canvas, x1, y1, x2, y2 int, cell canvas.Cell) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			c.SetCell(x, y, cell)
		}
	}
}
