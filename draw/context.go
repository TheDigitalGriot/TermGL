// Package draw provides 2D drawing primitives for the canvas.
package draw

import (
	"math"

	"github.com/charmbracelet/termgl/canvas"
)

// Transform2D represents a 2D affine transformation matrix.
// [a c e]
// [b d f]
// [0 0 1]
type Transform2D struct {
	A, B, C, D, E, F float64
}

// Identity returns the identity transformation.
func Identity() Transform2D {
	return Transform2D{A: 1, D: 1}
}

// Translate creates a translation transform.
func Translate(tx, ty float64) Transform2D {
	return Transform2D{A: 1, D: 1, E: tx, F: ty}
}

// Scale creates a scale transform.
func Scale(sx, sy float64) Transform2D {
	return Transform2D{A: sx, D: sy}
}

// ScaleUniform creates a uniform scale transform.
func ScaleUniform(s float64) Transform2D {
	return Transform2D{A: s, D: s}
}

// Rotate creates a rotation transform (angle in radians).
func Rotate(angle float64) Transform2D {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Transform2D{A: cos, B: sin, C: -sin, D: cos}
}

// RotateDegrees creates a rotation transform (angle in degrees).
func RotateDegrees(angle float64) Transform2D {
	return Rotate(angle * math.Pi / 180)
}

// Skew creates a skew transform (angles in radians).
func Skew(ax, ay float64) Transform2D {
	return Transform2D{A: 1, B: math.Tan(ay), C: math.Tan(ax), D: 1}
}

// Multiply multiplies two transforms.
func (t Transform2D) Multiply(other Transform2D) Transform2D {
	return Transform2D{
		A: t.A*other.A + t.C*other.B,
		B: t.B*other.A + t.D*other.B,
		C: t.A*other.C + t.C*other.D,
		D: t.B*other.C + t.D*other.D,
		E: t.A*other.E + t.C*other.F + t.E,
		F: t.B*other.E + t.D*other.F + t.F,
	}
}

// TransformPoint applies the transform to a point.
func (t Transform2D) TransformPoint(x, y float64) (float64, float64) {
	return t.A*x + t.C*y + t.E, t.B*x + t.D*y + t.F
}

// Inverse returns the inverse of the transform.
func (t Transform2D) Inverse() Transform2D {
	det := t.A*t.D - t.B*t.C
	if det == 0 {
		return Identity()
	}
	invDet := 1 / det
	return Transform2D{
		A: t.D * invDet,
		B: -t.B * invDet,
		C: -t.C * invDet,
		D: t.A * invDet,
		E: (t.C*t.F - t.D*t.E) * invDet,
		F: (t.B*t.E - t.A*t.F) * invDet,
	}
}

// contextState holds the saved state for a DrawContext.
type contextState struct {
	transform Transform2D
	cell      canvas.Cell
	lineWidth int
}

// DrawContext provides a stateful drawing API with transform stack.
// Inspired by HTML5 Canvas 2D context.
type DrawContext struct {
	canvas    *canvas.Canvas
	transform Transform2D
	stack     []contextState
	cell      canvas.Cell
	lineWidth int
}

// NewDrawContext creates a new drawing context for the given canvas.
func NewDrawContext(c *canvas.Canvas) *DrawContext {
	return &DrawContext{
		canvas:    c,
		transform: Identity(),
		cell:      canvas.NewCell('█', canvas.White),
		lineWidth: 1,
	}
}

// Canvas returns the underlying canvas.
func (ctx *DrawContext) Canvas() *canvas.Canvas {
	return ctx.canvas
}

// Save saves the current state (transform, style) onto the stack.
func (ctx *DrawContext) Save() {
	ctx.stack = append(ctx.stack, contextState{
		transform: ctx.transform,
		cell:      ctx.cell,
		lineWidth: ctx.lineWidth,
	})
}

// Restore restores the previously saved state from the stack.
func (ctx *DrawContext) Restore() {
	if len(ctx.stack) == 0 {
		return
	}
	state := ctx.stack[len(ctx.stack)-1]
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
	ctx.transform = state.transform
	ctx.cell = state.cell
	ctx.lineWidth = state.lineWidth
}

// --- Transform methods ---

// ResetTransform resets the transform to identity.
func (ctx *DrawContext) ResetTransform() {
	ctx.transform = Identity()
}

// SetTransform replaces the current transform.
func (ctx *DrawContext) SetTransform(t Transform2D) {
	ctx.transform = t
}

// GetTransform returns the current transform.
func (ctx *DrawContext) GetTransform() Transform2D {
	return ctx.transform
}

// Transform multiplies the current transform by the given transform.
func (ctx *DrawContext) Transform(t Transform2D) {
	ctx.transform = ctx.transform.Multiply(t)
}

// Translate applies a translation.
func (ctx *DrawContext) Translate(tx, ty float64) {
	ctx.Transform(Translate(tx, ty))
}

// Scale applies a scale.
func (ctx *DrawContext) Scale(sx, sy float64) {
	ctx.Transform(Scale(sx, sy))
}

// Rotate applies a rotation (radians).
func (ctx *DrawContext) Rotate(angle float64) {
	ctx.Transform(Rotate(angle))
}

// RotateDegrees applies a rotation (degrees).
func (ctx *DrawContext) RotateDegrees(angle float64) {
	ctx.Transform(RotateDegrees(angle))
}

// --- Style methods ---

// SetCell sets the current drawing cell (character + color).
func (ctx *DrawContext) SetCell(cell canvas.Cell) {
	ctx.cell = cell
}

// SetColor sets the foreground color.
func (ctx *DrawContext) SetColor(color canvas.Color) {
	ctx.cell.Foreground = color
	ctx.cell.HasFg = true
}

// SetChar sets the drawing character.
func (ctx *DrawContext) SetChar(r rune) {
	ctx.cell.Rune = r
}

// SetLineWidth sets the line width (for supported primitives).
func (ctx *DrawContext) SetLineWidth(width int) {
	if width < 1 {
		width = 1
	}
	ctx.lineWidth = width
}

// --- Drawing methods (all apply the current transform) ---

// DrawPoint draws a point at (x, y).
func (ctx *DrawContext) DrawPoint(x, y float64) {
	tx, ty := ctx.transform.TransformPoint(x, y)
	ix, iy := int(math.Round(tx)), int(math.Round(ty))
	ctx.canvas.SetCell(ix, iy, ctx.cell)
}

// DrawLine draws a line from (x1, y1) to (x2, y2).
func (ctx *DrawContext) DrawLine(x1, y1, x2, y2 float64) {
	tx1, ty1 := ctx.transform.TransformPoint(x1, y1)
	tx2, ty2 := ctx.transform.TransformPoint(x2, y2)
	LineFloat(ctx.canvas, tx1, ty1, tx2, ty2, ctx.cell)
}

// DrawRect draws a rectangle outline.
func (ctx *DrawContext) DrawRect(x, y, w, h float64) {
	// Transform all four corners
	x1, y1 := ctx.transform.TransformPoint(x, y)
	x2, y2 := ctx.transform.TransformPoint(x+w, y)
	x3, y3 := ctx.transform.TransformPoint(x+w, y+h)
	x4, y4 := ctx.transform.TransformPoint(x, y+h)

	LineFloat(ctx.canvas, x1, y1, x2, y2, ctx.cell)
	LineFloat(ctx.canvas, x2, y2, x3, y3, ctx.cell)
	LineFloat(ctx.canvas, x3, y3, x4, y4, ctx.cell)
	LineFloat(ctx.canvas, x4, y4, x1, y1, ctx.cell)
}

// FillRect fills a rectangle.
func (ctx *DrawContext) FillRect(x, y, w, h float64) {
	// For axis-aligned transforms, use optimized fill
	// For rotated/skewed, use scanline approach
	x1, y1 := ctx.transform.TransformPoint(x, y)
	x2, y2 := ctx.transform.TransformPoint(x+w, y+h)

	// Simple case: axis-aligned rectangle
	if ctx.transform.B == 0 && ctx.transform.C == 0 {
		FillRect(ctx.canvas,
			int(math.Round(math.Min(x1, x2))),
			int(math.Round(math.Min(y1, y2))),
			int(math.Round(math.Max(x1, x2))),
			int(math.Round(math.Max(y1, y2))),
			ctx.cell)
		return
	}

	// General case: draw as two triangles
	x3, y3 := ctx.transform.TransformPoint(x+w, y)
	x4, y4 := ctx.transform.TransformPoint(x, y+h)

	FillTriangleFloat(ctx.canvas, x1, y1, x3, y3, x2, y2, ctx.cell)
	FillTriangleFloat(ctx.canvas, x1, y1, x4, y4, x2, y2, ctx.cell)
}

// DrawTriangle draws a triangle outline.
func (ctx *DrawContext) DrawTriangle(x1, y1, x2, y2, x3, y3 float64) {
	tx1, ty1 := ctx.transform.TransformPoint(x1, y1)
	tx2, ty2 := ctx.transform.TransformPoint(x2, y2)
	tx3, ty3 := ctx.transform.TransformPoint(x3, y3)
	TriangleFloat(ctx.canvas, tx1, ty1, tx2, ty2, tx3, ty3, ctx.cell)
}

// FillTriangle fills a triangle.
func (ctx *DrawContext) FillTriangle(x1, y1, x2, y2, x3, y3 float64) {
	tx1, ty1 := ctx.transform.TransformPoint(x1, y1)
	tx2, ty2 := ctx.transform.TransformPoint(x2, y2)
	tx3, ty3 := ctx.transform.TransformPoint(x3, y3)
	FillTriangleFloat(ctx.canvas, tx1, ty1, tx2, ty2, tx3, ty3, ctx.cell)
}

// DrawCircle draws a circle outline.
func (ctx *DrawContext) DrawCircle(cx, cy, r float64) {
	// Transform center
	tcx, tcy := ctx.transform.TransformPoint(cx, cy)
	// Approximate radius (may be skewed)
	rx, _ := ctx.transform.TransformPoint(cx+r, cy)
	_, ry := ctx.transform.TransformPoint(cx, cy+r)
	avgR := (math.Abs(rx-tcx) + math.Abs(ry-tcy)) / 2

	Circle(ctx.canvas, int(math.Round(tcx)), int(math.Round(tcy)), int(math.Round(avgR)), ctx.cell)
}

// FillCircle fills a circle.
func (ctx *DrawContext) FillCircle(cx, cy, r float64) {
	tcx, tcy := ctx.transform.TransformPoint(cx, cy)
	rx, _ := ctx.transform.TransformPoint(cx+r, cy)
	_, ry := ctx.transform.TransformPoint(cx, cy+r)
	avgR := (math.Abs(rx-tcx) + math.Abs(ry-tcy)) / 2

	FillCircle(ctx.canvas, int(math.Round(tcx)), int(math.Round(tcy)), int(math.Round(avgR)), ctx.cell)
}

// DrawEllipse draws an ellipse outline.
func (ctx *DrawContext) DrawEllipse(cx, cy, rx, ry float64) {
	tcx, tcy := ctx.transform.TransformPoint(cx, cy)
	trx, _ := ctx.transform.TransformPoint(cx+rx, cy)
	_, try := ctx.transform.TransformPoint(cx, cy+ry)

	Ellipse(ctx.canvas,
		int(math.Round(tcx)), int(math.Round(tcy)),
		int(math.Round(math.Abs(trx-tcx))), int(math.Round(math.Abs(try-tcy))),
		ctx.cell)
}

// FillEllipse fills an ellipse.
func (ctx *DrawContext) FillEllipse(cx, cy, rx, ry float64) {
	tcx, tcy := ctx.transform.TransformPoint(cx, cy)
	trx, _ := ctx.transform.TransformPoint(cx+rx, cy)
	_, try := ctx.transform.TransformPoint(cx, cy+ry)

	FillEllipse(ctx.canvas,
		int(math.Round(tcx)), int(math.Round(tcy)),
		int(math.Round(math.Abs(trx-tcx))), int(math.Round(math.Abs(try-tcy))),
		ctx.cell)
}

// DrawPolyline draws a series of connected lines.
func (ctx *DrawContext) DrawPolyline(points [][2]float64) {
	if len(points) < 2 {
		return
	}
	for i := 1; i < len(points); i++ {
		ctx.DrawLine(points[i-1][0], points[i-1][1], points[i][0], points[i][1])
	}
}

// DrawPolygon draws a closed polygon outline.
func (ctx *DrawContext) DrawPolygon(points [][2]float64) {
	if len(points) < 3 {
		return
	}
	ctx.DrawPolyline(points)
	ctx.DrawLine(points[len(points)-1][0], points[len(points)-1][1], points[0][0], points[0][1])
}

// DrawText draws text at the given position.
func (ctx *DrawContext) DrawText(x, y float64, text string) {
	tx, ty := ctx.transform.TransformPoint(x, y)
	PutString(ctx.canvas, int(math.Round(tx)), int(math.Round(ty)), text, ctx.cell)
}

// Clear clears the canvas.
func (ctx *DrawContext) Clear() {
	ctx.canvas.Clear()
}

// ClearRect clears a rectangular region.
func (ctx *DrawContext) ClearRect(x, y, w, h float64) {
	x1, y1 := ctx.transform.TransformPoint(x, y)
	x2, y2 := ctx.transform.TransformPoint(x+w, y+h)

	ClearRect(ctx.canvas,
		int(math.Round(math.Min(x1, x2))),
		int(math.Round(math.Min(y1, y2))),
		int(math.Round(math.Max(x1, x2))),
		int(math.Round(math.Max(y1, y2))))
}

// MeasureText returns the width of text in cells.
func (ctx *DrawContext) MeasureText(text string) int {
	return MeasureString(text)
}

// Width returns the canvas width.
func (ctx *DrawContext) Width() int {
	return ctx.canvas.Width()
}

// Height returns the canvas height.
func (ctx *DrawContext) Height() int {
	return ctx.canvas.Height()
}
