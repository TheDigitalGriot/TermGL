// Spinning cube rendered with half-block pixels (▀) for true color shading.
//
// This example bridges the 3D pipeline (gl.NewCube, gl.Camera, math.Mat4) with
// the pixel-based framebuffer path (framebuffer.Framebuffer + encode.HalfBlockEncoder).
// Each terminal cell displays two vertical pixels via the ▀ character with
// independent foreground and background colors, giving double vertical resolution.
//
// Run with: go run ./examples/halfblock
package main

import (
	"fmt"
	"image/color"
	stdmath "math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl/anim"
	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/framebuffer"
	"github.com/charmbracelet/termgl/gl"
	"github.com/charmbracelet/termgl/math"
)

const (
	fps       = 30
	frequency = 5.0
	damping   = 0.5
)

type tickMsg time.Time

type model struct {
	// Rendering
	fb   *framebuffer.Framebuffer
	zbuf []float64

	// 3D scene
	camera   *gl.Camera
	cube     *gl.Mesh
	dirLight *gl.DirectionalLight
	ambLight *gl.AmbientLight

	// Animation
	rotX *anim.AnimatedFloat
	rotY *anim.AnimatedFloat

	// State
	autoRotate bool
	angle      float64

	// Dimensions
	vw, vh       int // virtual pixel resolution
	gridW, gridH int // terminal cell dimensions
	renderH      int // rows used for rendering (gridH - 1 for help text)
}

func initialModel() model {
	caps := detect.Capabilities()

	gridW := caps.GridSize.X
	gridH := caps.GridSize.Y
	renderH := gridH - 1 // reserve last row for help text
	vw := gridW
	vh := renderH * 2 // half-block: 2 vertical pixels per cell row

	fb := framebuffer.New(framebuffer.WithFixedSize(vw, vh))
	zbuf := make([]float64, vw*vh)

	cam := gl.NewPerspectiveCamera(30, float64(vw)/float64(vh), 0.1, 1000)
	cam.SetPosition(0, 0, 8)
	cam.LookAt(math.Vec3{})

	cube := gl.NewCube()

	dirLight := gl.NewDirectionalLight(math.Vec3{X: 0.5, Y: 0.5, Z: -1}, 1.0)
	ambLight := gl.NewAmbientLight(0.2)

	rotX := anim.NewAnimatedFloat(0, frequency, damping, fps)
	rotY := anim.NewAnimatedFloat(0, frequency, damping, fps)

	return model{
		fb:         fb,
		zbuf:       zbuf,
		camera:     cam,
		cube:       cube,
		dirLight:   dirLight,
		ambLight:   ambLight,
		rotX:       rotX,
		rotY:       rotY,
		autoRotate: true,
		vw:         vw,
		vh:         vh,
		gridW:      gridW,
		gridH:      gridH,
		renderH:    renderH,
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "left":
			m.rotY.Set(m.rotY.Target() - 45)
			m.autoRotate = false
		case "right":
			m.rotY.Set(m.rotY.Target() + 45)
			m.autoRotate = false
		case "up":
			m.rotX.Set(m.rotX.Target() - 45)
			m.autoRotate = false
		case "down":
			m.rotX.Set(m.rotX.Target() + 45)
			m.autoRotate = false
		case "space":
			m.autoRotate = !m.autoRotate
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.gridW = msg.Width
		m.gridH = msg.Height
		m.renderH = m.gridH - 1
		if m.renderH < 1 {
			m.renderH = 1
		}
		m.vw = m.gridW
		m.vh = m.renderH * 2
		m.fb.Resize(m.vw, m.vh)
		m.zbuf = make([]float64, m.vw*m.vh)
		m.camera.SetAspect(float64(m.vw) / float64(m.vh))
		return m, nil

	case tickMsg:
		if m.autoRotate {
			m.angle += 1.5
			m.rotY.Set(m.angle)
			m.rotX.Set(m.angle * 0.3)
		}
		m.rotX.Update()
		m.rotY.Update()
		m.cube.SetRotation(m.rotX.Get(), m.rotY.Get(), 0)
		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	// Clear framebuffer and z-buffer
	m.fb.Clear(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	clearZBuf(m.zbuf)

	// Render the cube
	renderScene(&m)

	// Encode framebuffer to half-block string for Bubble Tea
	output := framebufferToString(m.fb, m.gridW, m.renderH)

	help := "\n[arrows] rotate  [space] toggle auto  [q] quit"
	return output + help
}

// framebufferToString converts the framebuffer to a multi-line string of ▀
// characters with ANSI 24-bit color codes. Each terminal row maps to 2 pixel
// rows: foreground = top pixel, background = bottom pixel.
func framebufferToString(fb *framebuffer.Framebuffer, gridW, renderH int) string {
	var buf strings.Builder
	// Pre-allocate: ~30 bytes per cell is a reasonable estimate
	buf.Grow(gridW * renderH * 30)

	var lastFG, lastBG color.RGBA
	for row := 0; row < renderH; row++ {
		firstInRow := true
		for col := 0; col < gridW; col++ {
			topY := row * 2
			botY := row*2 + 1

			var top, bot color.RGBA
			if topY < fb.Height && col < fb.Width {
				top = fb.Pixels.RGBAAt(col, topY)
			}
			if botY < fb.Height && col < fb.Width {
				bot = fb.Pixels.RGBAAt(col, botY)
			}

			if firstInRow || top != lastFG {
				fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm", top.R, top.G, top.B)
				lastFG = top
			}
			if firstInRow || bot != lastBG {
				fmt.Fprintf(&buf, "\x1b[48;2;%d;%d;%dm", bot.R, bot.G, bot.B)
				lastBG = bot
			}
			buf.WriteString("▀")
			firstInRow = false
		}
		buf.WriteString("\x1b[0m")
		lastFG = color.RGBA{}
		lastBG = color.RGBA{}
		if row < renderH-1 {
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

// clearZBuf resets the z-buffer to maximum depth.
func clearZBuf(zbuf []float64) {
	for i := range zbuf {
		zbuf[i] = stdmath.MaxFloat64
	}
}

// renderScene projects and rasterizes the cube into the framebuffer.
func renderScene(m *model) {
	modelMatrix := m.cube.Transform.Matrix()
	viewProjMatrix := m.camera.ViewProjectionMatrix()
	mvpMatrix := viewProjMatrix.Mul(modelMatrix)

	halfW := float64(m.vw) / 2.0
	halfH := float64(m.vh) / 2.0

	for _, tri := range m.cube.Triangles {
		// Backface culling
		worldNormal := modelMatrix.MulVec3Dir(tri.FaceNormal).Normalize()
		worldPos := modelMatrix.MulVec3(tri.Vertices[0].Position)
		viewDir := worldPos.Sub(m.camera.Position)
		if worldNormal.Dot(viewDir) >= 0 {
			continue
		}

		// Project vertices to screen space
		// MulVec3 returns NDC (with perspective divide). The perspective matrix
		// already accounts for aspect ratio, so map NDC [-1,1] to full pixel range.
		// Half-block pixels are roughly square (1 cell wide × 0.5 cell tall ≈ 8×8px).
		var screenVerts [3]math.Vec3
		visible := true
		for i, vert := range tri.Vertices {
			ndc := mvpMatrix.MulVec3(vert.Position)
			if ndc.Z < -10 {
				visible = false
				break
			}
			screenVerts[i] = math.Vec3{
				X: (ndc.X + 1.0) * halfW,
				Y: (1.0 - ndc.Y) * halfH,
				Z: ndc.Z,
			}
		}
		if !visible {
			continue
		}

		// Flat shading: compute lighting intensity
		diffuse := stdmath.Abs(worldNormal.Dot(m.dirLight.Direction)) * m.dirLight.Intensity
		intensity := diffuse*(1-m.ambLight.Intensity) + m.ambLight.Intensity
		if intensity > 1 {
			intensity = 1
		}
		if intensity < 0 {
			intensity = 0
		}

		// Per-face base color
		br, bg, bb := faceBaseColor(tri.FaceNormal)
		faceColor := color.RGBA{
			R: uint8(br * intensity),
			G: uint8(bg * intensity),
			B: uint8(bb * intensity),
			A: 255,
		}

		rasterizeTriangle(m.fb, m.zbuf, m.vw, m.vh, screenVerts, faceColor)
	}
}

// faceBaseColor returns a distinct base color for each cube face axis.
func faceBaseColor(normal math.Vec3) (float64, float64, float64) {
	ax := stdmath.Abs(normal.X)
	ay := stdmath.Abs(normal.Y)
	az := stdmath.Abs(normal.Z)
	switch {
	case ax >= ay && ax >= az:
		return 230, 100, 100 // red-ish (left/right)
	case ay >= ax && ay >= az:
		return 100, 200, 100 // green-ish (top/bottom)
	default:
		return 100, 149, 237 // blue-ish (front/back)
	}
}

// ============================================================================
// Scanline triangle rasterizer (outputs RGBA pixels to framebuffer)
// Ported from gl/rasterizer.go
// ============================================================================

func rasterizeTriangle(fb *framebuffer.Framebuffer, zbuf []float64, vw, vh int, verts [3]math.Vec3, col color.RGBA) {
	// Z-buffer plane equation
	e1 := verts[1].Sub(verts[0])
	e2 := verts[2].Sub(verts[0])
	zCross := e1.Cross(e2)
	zVert := verts[0]

	// Sort vertices by Y ascending
	sorted := verts
	if sorted[0].Y > sorted[1].Y {
		sorted[0], sorted[1] = sorted[1], sorted[0]
	}
	if sorted[1].Y > sorted[2].Y {
		sorted[1], sorted[2] = sorted[2], sorted[1]
	}
	if sorted[0].Y > sorted[1].Y {
		sorted[0], sorted[1] = sorted[1], sorted[0]
	}

	if sorted[1].Y == sorted[2].Y {
		// Flat bottom
		fillFlatBottom(fb, zbuf, vw, vh, sorted[0], sorted[1], sorted[2], zCross, zVert, col)
	} else if sorted[0].Y == sorted[1].Y {
		// Flat top
		fillFlatTop(fb, zbuf, vw, vh, sorted[0], sorted[1], sorted[2], zCross, zVert, col)
	} else {
		// General case: split into flat-bottom and flat-top.
		// Use parametric interpolation along edge 0→2 to find X at sorted[1].Y.
		t := (sorted[1].Y - sorted[0].Y) / (sorted[2].Y - sorted[0].Y)
		newX := sorted[0].X + t*(sorted[2].X-sorted[0].X)
		newZ := math.CalcZ(newX, sorted[1].Y, zCross, zVert)
		newVert := math.Vec3{X: newX, Y: sorted[1].Y, Z: newZ}

		fillFlatBottom(fb, zbuf, vw, vh, sorted[0], newVert, sorted[1], zCross, zVert, col)
		fillFlatTop(fb, zbuf, vw, vh, newVert, sorted[1], sorted[2], zCross, zVert, col)
	}
}

// fillFlatBottom fills a triangle where v1.Y == v2.Y (flat bottom edge).
// v0 is the top vertex. Uses incremental inverse-slope stepping.
func fillFlatBottom(fb *framebuffer.Framebuffer, zbuf []float64, vw, vh int, v0, v1, v2, zCross, zVert math.Vec3, col color.RGBA) {
	dy := v1.Y - v0.Y
	if dy == 0 {
		return
	}
	invSlope1 := (v1.X - v0.X) / dy
	invSlope2 := (v2.X - v0.X) / dy

	curX1 := v0.X
	curX2 := v0.X

	startY := int(stdmath.Round(v0.Y))
	endY := int(stdmath.Round(v1.Y))

	for y := startY; y <= endY; y++ {
		drawHLine(fb, zbuf, vw, vh, int(stdmath.Round(curX1)), int(stdmath.Round(curX2)), y, zCross, zVert, col)
		curX1 += invSlope1
		curX2 += invSlope2
	}
}

// fillFlatTop fills a triangle where v0.Y == v1.Y (flat top edge).
// v2 is the bottom vertex. Uses incremental inverse-slope stepping.
func fillFlatTop(fb *framebuffer.Framebuffer, zbuf []float64, vw, vh int, v0, v1, v2, zCross, zVert math.Vec3, col color.RGBA) {
	dy := v2.Y - v0.Y
	if dy == 0 {
		return
	}
	invSlope1 := (v2.X - v0.X) / dy
	invSlope2 := (v2.X - v1.X) / dy

	curX1 := v2.X
	curX2 := v2.X

	startY := int(stdmath.Round(v2.Y))
	endY := int(stdmath.Round(v0.Y))

	for y := startY; y >= endY; y-- {
		drawHLine(fb, zbuf, vw, vh, int(stdmath.Round(curX1)), int(stdmath.Round(curX2)), y, zCross, zVert, col)
		curX1 -= invSlope1
		curX2 -= invSlope2
	}
}

func drawHLine(fb *framebuffer.Framebuffer, zbuf []float64, vw, vh, x1, x2, y int, zCross, zVert math.Vec3, col color.RGBA) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		drawPixel(fb, zbuf, vw, vh, x, y, zCross, zVert, col)
	}
}

func drawPixel(fb *framebuffer.Framebuffer, zbuf []float64, vw, vh, x, y int, zCross, zVert math.Vec3, col color.RGBA) {
	if x < 0 || x >= vw || y < 0 || y >= vh {
		return
	}
	depth := math.CalcZ(float64(x), float64(y), zCross, zVert)
	idx := y*vw + x
	if depth >= zbuf[idx] {
		return
	}
	zbuf[idx] = depth
	fb.SetPixel(x, y, col)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
