// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"math"

	"github.com/charmbracelet/termgl/canvas"
)

// Triangle draws a triangle outline using three vertices.
func Triangle(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 int, cell canvas.Cell) {
	Line(c, x1, y1, x2, y2, cell)
	Line(c, x2, y2, x3, y3, cell)
	Line(c, x3, y3, x1, y1, cell)
}

// TriangleFloat draws a triangle outline using float coordinates.
func TriangleFloat(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 float64, cell canvas.Cell) {
	Triangle(c,
		int(math.Round(x1)), int(math.Round(y1)),
		int(math.Round(x2)), int(math.Round(y2)),
		int(math.Round(x3)), int(math.Round(y3)),
		cell)
}

// FillTriangle fills a triangle using scanline rasterization.
// Uses flat-top/flat-bottom triangle decomposition algorithm.
func FillTriangle(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 int, cell canvas.Cell) {
	// Sort vertices by Y coordinate (ascending)
	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	if y2 > y3 {
		x2, y2, x3, y3 = x3, y3, x2, y2
	}
	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}

	// Now y1 <= y2 <= y3

	// Degenerate case: all points on same horizontal line
	if y1 == y3 {
		minX := min(x1, min(x2, x3))
		maxX := max(x1, max(x2, x3))
		HLine(c, minX, maxX, y1, cell)
		return
	}

	// Check for flat triangles
	if y2 == y3 {
		// Flat bottom triangle
		fillFlatBottomTriangle(c, x1, y1, x2, y2, x3, y3, cell)
	} else if y1 == y2 {
		// Flat top triangle
		fillFlatTopTriangle(c, x1, y1, x2, y2, x3, y3, cell)
	} else {
		// General case: split into flat-bottom and flat-top triangles
		// Calculate the x-coordinate of the splitting point on the long edge
		x4 := int(math.Round(float64(x1) + (float64(y2-y1)/float64(y3-y1))*float64(x3-x1)))
		y4 := y2

		// Fill both sub-triangles
		fillFlatBottomTriangle(c, x1, y1, x2, y2, x4, y4, cell)
		fillFlatTopTriangle(c, x2, y2, x4, y4, x3, y3, cell)
	}
}

// FillTriangleFloat fills a triangle using float coordinates.
func FillTriangleFloat(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 float64, cell canvas.Cell) {
	FillTriangle(c,
		int(math.Round(x1)), int(math.Round(y1)),
		int(math.Round(x2)), int(math.Round(y2)),
		int(math.Round(x3)), int(math.Round(y3)),
		cell)
}

// fillFlatBottomTriangle fills a triangle where v2.y == v3.y (bottom edge is flat).
// v1 is the top vertex.
func fillFlatBottomTriangle(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 int, cell canvas.Cell) {
	// Calculate inverse slopes (change in x per unit y)
	invSlope1 := float64(x2-x1) / float64(y2-y1)
	invSlope2 := float64(x3-x1) / float64(y3-y1)

	// Start both edges at top vertex
	curX1 := float64(x1)
	curX2 := float64(x1)

	// Draw horizontal scanlines from top to bottom
	for y := y1; y <= y2; y++ {
		HLine(c, int(math.Round(curX1)), int(math.Round(curX2)), y, cell)
		curX1 += invSlope1
		curX2 += invSlope2
	}
}

// fillFlatTopTriangle fills a triangle where v1.y == v2.y (top edge is flat).
// v3 is the bottom vertex.
func fillFlatTopTriangle(c *canvas.Canvas, x1, y1, x2, y2, x3, y3 int, cell canvas.Cell) {
	// Calculate inverse slopes (change in x per unit y)
	invSlope1 := float64(x3-x1) / float64(y3-y1)
	invSlope2 := float64(x3-x2) / float64(y3-y2)

	// Start both edges at bottom vertex
	curX1 := float64(x3)
	curX2 := float64(x3)

	// Draw horizontal scanlines from bottom to top
	for y := y3; y >= y1; y-- {
		HLine(c, int(math.Round(curX1)), int(math.Round(curX2)), y, cell)
		curX1 -= invSlope1
		curX2 -= invSlope2
	}
}

// Circle draws a circle outline using Bresenham's circle algorithm.
func Circle(c *canvas.Canvas, cx, cy, r int, cell canvas.Cell) {
	x := 0
	y := r
	d := 3 - 2*r

	drawCirclePoints(c, cx, cy, x, y, cell)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		drawCirclePoints(c, cx, cy, x, y, cell)
	}
}

// drawCirclePoints draws the 8 symmetric points of a circle.
func drawCirclePoints(c *canvas.Canvas, cx, cy, x, y int, cell canvas.Cell) {
	c.SetCell(cx+x, cy+y, cell)
	c.SetCell(cx-x, cy+y, cell)
	c.SetCell(cx+x, cy-y, cell)
	c.SetCell(cx-x, cy-y, cell)
	c.SetCell(cx+y, cy+x, cell)
	c.SetCell(cx-y, cy+x, cell)
	c.SetCell(cx+y, cy-x, cell)
	c.SetCell(cx-y, cy-x, cell)
}

// FillCircle fills a circle using horizontal scanlines.
func FillCircle(c *canvas.Canvas, cx, cy, r int, cell canvas.Cell) {
	x := 0
	y := r
	d := 3 - 2*r

	fillCircleLines(c, cx, cy, x, y, cell)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		fillCircleLines(c, cx, cy, x, y, cell)
	}
}

// fillCircleLines draws horizontal lines for filled circle.
func fillCircleLines(c *canvas.Canvas, cx, cy, x, y int, cell canvas.Cell) {
	HLine(c, cx-x, cx+x, cy+y, cell)
	HLine(c, cx-x, cx+x, cy-y, cell)
	HLine(c, cx-y, cx+y, cy+x, cell)
	HLine(c, cx-y, cx+y, cy-x, cell)
}

// Ellipse draws an ellipse outline using the midpoint algorithm.
func Ellipse(c *canvas.Canvas, cx, cy, rx, ry int, cell canvas.Cell) {
	if rx == 0 && ry == 0 {
		c.SetCell(cx, cy, cell)
		return
	}
	if rx == 0 {
		VLine(c, cx, cy-ry, cy+ry, cell)
		return
	}
	if ry == 0 {
		HLine(c, cx-rx, cx+rx, cy, cell)
		return
	}

	rx2 := rx * rx
	ry2 := ry * ry
	twoRx2 := 2 * rx2
	twoRy2 := 2 * ry2
	x := 0
	y := ry
	px := 0
	py := twoRx2 * y

	// Region 1
	p := int(float64(ry2) - float64(rx2*ry) + 0.25*float64(rx2))
	for px < py {
		drawEllipsePoints(c, cx, cy, x, y, cell)
		x++
		px += twoRy2
		if p < 0 {
			p += ry2 + px
		} else {
			y--
			py -= twoRx2
			p += ry2 + px - py
		}
	}

	// Region 2
	p = int(float64(ry2)*float64(x*x+x)+0.25*float64(ry2) + float64(rx2*(y-1)*(y-1)) - float64(rx2*ry2))
	for y >= 0 {
		drawEllipsePoints(c, cx, cy, x, y, cell)
		y--
		py -= twoRx2
		if p > 0 {
			p += rx2 - py
		} else {
			x++
			px += twoRy2
			p += rx2 - py + px
		}
	}
}

// drawEllipsePoints draws the 4 symmetric points of an ellipse.
func drawEllipsePoints(c *canvas.Canvas, cx, cy, x, y int, cell canvas.Cell) {
	c.SetCell(cx+x, cy+y, cell)
	c.SetCell(cx-x, cy+y, cell)
	c.SetCell(cx+x, cy-y, cell)
	c.SetCell(cx-x, cy-y, cell)
}

// FillEllipse fills an ellipse using horizontal scanlines.
func FillEllipse(c *canvas.Canvas, cx, cy, rx, ry int, cell canvas.Cell) {
	if rx == 0 && ry == 0 {
		c.SetCell(cx, cy, cell)
		return
	}
	if rx == 0 {
		VLine(c, cx, cy-ry, cy+ry, cell)
		return
	}
	if ry == 0 {
		HLine(c, cx-rx, cx+rx, cy, cell)
		return
	}

	rx2 := rx * rx
	ry2 := ry * ry
	twoRx2 := 2 * rx2
	twoRy2 := 2 * ry2
	x := 0
	y := ry
	px := 0
	py := twoRx2 * y

	// Track last y to avoid drawing same line twice
	lastY := -1

	// Region 1
	p := int(float64(ry2) - float64(rx2*ry) + 0.25*float64(rx2))
	for px < py {
		if y != lastY {
			HLine(c, cx-x, cx+x, cy+y, cell)
			HLine(c, cx-x, cx+x, cy-y, cell)
			lastY = y
		}
		x++
		px += twoRy2
		if p < 0 {
			p += ry2 + px
		} else {
			y--
			py -= twoRx2
			p += ry2 + px - py
		}
	}

	// Region 2
	p = int(float64(ry2)*float64(x*x+x)+0.25*float64(ry2) + float64(rx2*(y-1)*(y-1)) - float64(rx2*ry2))
	for y >= 0 {
		HLine(c, cx-x, cx+x, cy+y, cell)
		HLine(c, cx-x, cx+x, cy-y, cell)
		y--
		py -= twoRx2
		if p > 0 {
			p += rx2 - py
		} else {
			x++
			px += twoRy2
			p += rx2 - py + px
		}
	}
}
