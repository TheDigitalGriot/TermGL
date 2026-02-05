// Demo application showcasing all TermGL features.
// Press 1-9 to switch between demos, Q to quit, Esc to return to menu.
package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl/anim"
	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/draw"
	"github.com/charmbracelet/termgl/gl"
	glmath "github.com/charmbracelet/termgl/math"
	"github.com/charmbracelet/termgl/svg"
)

const (
	width  = 80
	height = 24
)

type demoType int

const (
	demoMenu demoType = iota
	demoTriangles
	demoText
	demoGradients
	demoShaders
	demoTextures
	demoSubcell
	demoEasing
	demoTimeline
	demoSVG
	demoKitty
	demo3DCube
	demo3DWireframe
	demo3DShading

	// TermGL-C-Plus ports (z,x,c,v,b,n,m keys)
	demoTeapot       // z - Utah Teapot (STL loading + lighting)
	demoColorPalette // x - Color palette reference
	demoMandelbrot   // c - Mandelbrot fractal zoom
	demoKeyboardDemo // v - Real-time keyboard input
	demoTexturedCube // b - Textured cube with UV mapping (different from demoTextures)
	demoRGBCircles   // n - 24-bit RGB gradient circles
	demoMouseDemo    // m - Mouse position/button tracking
)

type tickMsg time.Time

type model struct {
	canvas      *canvas.Canvas
	currentDemo demoType
	frame       int
	startTime   time.Time

	// Animation state
	tween    *anim.Tween
	timeline *anim.Timeline
	animX    float64
	animY    float64

	// Subcell canvas
	subcell *canvas.SubCellCanvas

	// Kitty backend
	kitty *canvas.KittyBackend

	// 3D rendering
	renderer    *gl.Renderer
	camera      *gl.Camera
	cubeMesh    *gl.Mesh
	pyramidMesh *gl.Mesh

	// TermGL-C-Plus demo state
	teapotMesh     *gl.Mesh    // Utah teapot STL mesh
	cubeTexture    *gl.Texture // Texture for textured cube demo
	lastKey        string      // Last pressed key for keyboard demo
	mouseX         int         // Mouse X position
	mouseY         int         // Mouse Y position
	mouseButton    string      // Last mouse button action
	mandelbrotZoom float64     // Current zoom level for mandelbrot
}

func initialModel() model {
	c := canvas.New(width, height)
	return model{
		canvas:      c,
		currentDemo: demoMenu,
		startTime:   time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.EnterAltScreen)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse events for mouse demo
		if m.currentDemo == demoMouseDemo {
			m.mouseX = msg.X
			m.mouseY = msg.Y
			switch msg.Button {
			case tea.MouseButtonLeft:
				if msg.Action == tea.MouseActionPress {
					m.mouseButton = "Left Click"
				} else if msg.Action == tea.MouseActionRelease {
					m.mouseButton = "Left Release"
				}
			case tea.MouseButtonRight:
				if msg.Action == tea.MouseActionPress {
					m.mouseButton = "Right Click"
				} else if msg.Action == tea.MouseActionRelease {
					m.mouseButton = "Right Release"
				}
			case tea.MouseButtonMiddle:
				if msg.Action == tea.MouseActionPress {
					m.mouseButton = "Middle Click"
				} else if msg.Action == tea.MouseActionRelease {
					m.mouseButton = "Middle Release"
				}
			case tea.MouseButtonWheelUp:
				m.mouseButton = "Wheel Up"
			case tea.MouseButtonWheelDown:
				m.mouseButton = "Wheel Down"
			default:
				if msg.Action == tea.MouseActionMotion {
					m.mouseButton = "Motion"
				}
			}
			return m, nil
		}

	case tea.KeyMsg:
		// Capture key for keyboard demo (except navigation keys)
		if m.currentDemo == demoKeyboardDemo {
			key := msg.String()
			if key != "esc" && key != "q" && key != "ctrl+c" {
				m.lastKey = key
				return m, nil
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.currentDemo = demoMenu
			m.frame = 0
			// Disable mouse tracking when returning to menu
			return m, tea.DisableMouse
		case "1":
			m.currentDemo = demoTriangles
			m.frame = 0
			return m, nil
		case "2":
			m.currentDemo = demoText
			m.frame = 0
			return m, nil
		case "3":
			m.currentDemo = demoGradients
			m.frame = 0
			return m, nil
		case "4":
			m.currentDemo = demoShaders
			m.frame = 0
			return m, nil
		case "5":
			m.currentDemo = demoTextures
			m.frame = 0
			return m, nil
		case "6":
			m.currentDemo = demoSubcell
			m.frame = 0
			m.subcell = canvas.NewSubCellCanvas(m.canvas, canvas.SubCellBraille)
			return m, nil
		case "7":
			m.currentDemo = demoEasing
			m.frame = 0
			m.tween = anim.NewTween(5, 70, 2*time.Second).
				Ease(anim.EaseOutBounce).
				Repeat(-1).
				Yoyo(true).
				Start()
			return m, nil
		case "8":
			m.currentDemo = demoTimeline
			m.frame = 0
			return m, nil
		case "9":
			m.currentDemo = demoSVG
			m.frame = 0
			return m, nil
		case "0":
			m.currentDemo = demoKitty
			m.frame = 0
			m.kitty = canvas.NewKittyBackend(160, 96) // 2x resolution
			return m, nil
		case "a", "A":
			m.currentDemo = demo3DCube
			m.frame = 0
			m.setup3D()
			return m, nil
		case "B": // Uppercase only - 3D Wireframe
			m.currentDemo = demo3DWireframe
			m.frame = 0
			m.setup3D()
			return m, nil
		case "C": // Uppercase only - 3D Shading
			m.currentDemo = demo3DShading
			m.frame = 0
			m.setup3D()
			return m, nil

		// TermGL-C-Plus ports (z,x,c,v,b,n,m)
		case "z", "Z":
			m.currentDemo = demoTeapot
			m.frame = 0
			m.setupTeapot()
			return m, nil
		case "x", "X":
			m.currentDemo = demoColorPalette
			m.frame = 0
			return m, nil
		case "c": // Lowercase only - Mandelbrot
			m.currentDemo = demoMandelbrot
			m.frame = 0
			m.mandelbrotZoom = 0
			return m, nil
		case "v", "V":
			m.currentDemo = demoKeyboardDemo
			m.frame = 0
			m.lastKey = ""
			return m, nil
		case "b": // Lowercase only - Textured Cube
			m.currentDemo = demoTexturedCube
			m.frame = 0
			m.setup3D()
			return m, nil
		case "n", "N":
			m.currentDemo = demoRGBCircles
			m.frame = 0
			return m, nil
		case "m", "M":
			m.currentDemo = demoMouseDemo
			m.frame = 0
			m.mouseButton = "none"
			return m, tea.EnableMouseAllMotion
		}

	case tickMsg:
		m.frame++

		// Update animations
		if m.tween != nil && m.currentDemo == demoEasing {
			m.tween.Update()
		}
		if m.timeline != nil && m.currentDemo == demoTimeline {
			m.timeline.Update()
		}

		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	m.canvas.Clear()

	switch m.currentDemo {
	case demoMenu:
		m.renderMenu()
	case demoTriangles:
		m.renderTriangles()
	case demoText:
		m.renderText()
	case demoGradients:
		m.renderGradients()
	case demoShaders:
		m.renderShaders()
	case demoTextures:
		m.renderTextures()
	case demoSubcell:
		m.renderSubcell()
	case demoEasing:
		m.renderEasing()
	case demoTimeline:
		m.renderTimeline()
	case demoSVG:
		m.renderSVG()
	case demoKitty:
		m.renderKitty()
	case demo3DCube:
		m.render3DCube()
	case demo3DWireframe:
		m.render3DWireframe()
	case demo3DShading:
		m.render3DShading()

	// TermGL-C-Plus demos
	case demoTeapot:
		m.renderTeapot()
	case demoColorPalette:
		m.renderColorPalette()
	case demoMandelbrot:
		m.renderMandelbrot()
	case demoKeyboardDemo:
		m.renderKeyboardDemo()
	case demoTexturedCube:
		m.renderTexturedCube()
	case demoRGBCircles:
		m.renderRGBCircles()
	case demoMouseDemo:
		m.renderMouseDemo()
	}

	return m.canvas.String()
}

func (m *model) renderMenu() {
	title := "TermGL Feature Demo"
	draw.PutStringCentered(m.canvas, 1, title, canvas.NewCell(' ', canvas.White))

	menuStyle := canvas.NewCell(' ', canvas.Cyan)
	termglStyle := canvas.NewCell(' ', canvas.Yellow)

	items := []string{
		"--- TermGL Go Demos ---",
		"[1] 2D Triangles    [A] 3D Cube",
		"[2] Text            [B] 3D Wireframe",
		"[3] Gradients       [C] 3D Shading",
		"[4] Shaders",
		"[5] Textures",
		"[6] Subcell",
		"[7] Easing",
		"[8] Timeline",
		"[9] SVG Paths",
		"[0] Kitty Protocol",
	}

	termglCItems := []string{
		"--- TermGL-C-Plus Ports ---",
		"[Z] Utah Teapot     [V] Keyboard",
		"[X] Color Palette   [B] Textured Cube",
		"[C] Mandelbrot      [N] RGB Circles",
		"                    [M] Mouse Tracking",
	}

	startY := 3
	for i, item := range items {
		draw.PutStringCentered(m.canvas, startY+i, item, menuStyle)
	}

	startY = startY + len(items) + 1
	for i, item := range termglCItems {
		draw.PutStringCentered(m.canvas, startY+i, item, termglStyle)
	}

	draw.PutStringCentered(m.canvas, height-2, "[Q] Quit  [Esc] Back to Menu", canvas.NewCell(' ', canvas.White))

	// Animated border
	t := float64(m.frame) / 30.0
	borderCell := canvas.NewCell('*', canvas.Yellow)
	for x := 0; x < width; x++ {
		intensity := (math.Sin(float64(x)*0.2+t*3) + 1) / 2
		if intensity > 0.7 {
			m.canvas.SetCell(x, 0, borderCell)
			m.canvas.SetCell(x, height-1, borderCell)
		}
	}
}

func (m *model) renderTriangles() {
	draw.PutString(m.canvas, 2, 1, "Demo 1: 2D Triangles & Shapes", canvas.NewCell(' ', canvas.Yellow))

	// Animated rotation
	t := float64(m.frame) / 30.0
	cx, cy := 20.0, 12.0
	r := 8.0

	// Rotating triangle
	for i := 0; i < 3; i++ {
		angle := t + float64(i)*2*math.Pi/3
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)*0.5 // Aspect ratio compensation
		if i == 0 {
			draw.FillTriangleFloat(m.canvas, cx, cy,
				cx+r*math.Cos(t), cy+r*math.Sin(t)*0.5,
				cx+r*math.Cos(t+2*math.Pi/3), cy+r*math.Sin(t+2*math.Pi/3)*0.5,
				canvas.NewCell('▲', canvas.Cyan))
		}
		_ = x
		_ = y
	}

	// Circle
	draw.FillCircle(m.canvas, 45, 12, 6, canvas.NewCell('●', canvas.Green))
	draw.Circle(m.canvas, 45, 12, 8, canvas.NewCell('○', canvas.White))

	// Ellipse
	draw.FillEllipse(m.canvas, 65, 12, 10, 5, canvas.NewCell('◆', canvas.Magenta))

	// Rectangle
	draw.Rect(m.canvas, 2, 18, 25, 22, canvas.NewCell('░', canvas.Blue))
	draw.FillRect(m.canvas, 4, 19, 23, 21, canvas.NewCell('▓', canvas.Blue))

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderText() {
	draw.PutString(m.canvas, 2, 1, "Demo 2: Text Rendering", canvas.NewCell(' ', canvas.Yellow))

	// Basic text
	draw.PutStringColor(m.canvas, 2, 4, "PutStringColor: Hello World!", canvas.Red)

	// Centered text
	draw.PutStringCentered(m.canvas, 6, "This text is centered", canvas.NewCell(' ', canvas.Cyan))

	// Right-aligned text
	draw.PutStringRight(m.canvas, width-3, 8, "Right aligned", canvas.NewCell(' ', canvas.Green))

	// Vertical text
	draw.PutStringVertical(m.canvas, 2, 10, "VERTICAL", canvas.NewCell(' ', canvas.Magenta))

	// Styled text with per-character styling
	rainbow := []canvas.Color{canvas.Red, canvas.Yellow, canvas.Green, canvas.Cyan, canvas.Blue, canvas.Magenta}
	text := "Rainbow styled text!"
	for i, r := range text {
		color := rainbow[i%len(rainbow)]
		m.canvas.SetCell(10+i, 12, canvas.NewCell(r, color))
	}

	// Wrapped text
	longText := "This is a long text that will wrap automatically when it reaches the maximum width specified."
	draw.PutStringWrap(m.canvas, 10, 15, 30, longText, canvas.NewCell(' ', canvas.White))

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderGradients() {
	draw.PutString(m.canvas, 2, 1, "Demo 3: Gradients", canvas.NewCell(' ', canvas.Yellow))

	// Full gradient
	draw.PutString(m.canvas, 2, 3, "Full 70-char gradient:", canvas.NewCell(' ', canvas.White))
	for i := 0; i < 70 && i < width-4; i++ {
		intensity := uint8(i * 255 / 70)
		char := draw.GradientFull.Char(intensity)
		m.canvas.SetCell(2+i, 4, canvas.NewCell(char, canvas.White))
	}

	// Minimal gradient
	draw.PutString(m.canvas, 2, 6, "Minimal gradient:", canvas.NewCell(' ', canvas.White))
	for i := 0; i < 40; i++ {
		intensity := uint8(i * 255 / 40)
		char := draw.GradientMin.Char(intensity)
		m.canvas.SetCell(2+i, 7, canvas.NewCell(char, canvas.Cyan))
	}

	// Block gradient
	draw.PutString(m.canvas, 2, 9, "Block gradient:", canvas.NewCell(' ', canvas.White))
	for i := 0; i < 40; i++ {
		intensity := uint8(i * 255 / 40)
		char := draw.GradientBlocks.Char(intensity)
		m.canvas.SetCell(2+i, 10, canvas.NewCell(char, canvas.Green))
	}

	// Animated intensity bar
	draw.PutString(m.canvas, 2, 12, "Animated:", canvas.NewCell(' ', canvas.White))
	t := float64(m.frame) / 30.0
	for i := 0; i < 50; i++ {
		wave := (math.Sin(float64(i)*0.2+t*3) + 1) / 2
		intensity := uint8(wave * 255)
		char := draw.GradientFull.Char(intensity)
		m.canvas.SetCell(12+i, 12, canvas.NewCell(char, canvas.Yellow))
	}

	// Gradient sphere approximation
	draw.PutString(m.canvas, 2, 14, "Gradient sphere:", canvas.NewCell(' ', canvas.White))
	cx, cy := 25.0, 18.0
	r := 4.0
	for dy := -r; dy <= r; dy++ {
		for dx := -r * 2; dx <= r*2; dx++ {
			dist := math.Sqrt((dx/2)*(dx/2) + dy*dy)
			if dist <= r {
				intensity := uint8((1 - dist/r) * 255)
				char := draw.GradientFull.Char(intensity)
				m.canvas.SetCell(int(cx+dx), int(cy+dy), canvas.NewCell(char, canvas.White))
			}
		}
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderShaders() {
	draw.PutString(m.canvas, 2, 1, "Demo 4: Pixel Shaders", canvas.NewCell(' ', canvas.Yellow))

	// Simple gradient shader triangle
	v0 := draw.NewVertex2D(15, 5, 0, 0)
	v1 := draw.NewVertex2D(5, 18, 255, 0)
	v2 := draw.NewVertex2D(25, 18, 128, 255)

	gradShader := draw.GradientShader(draw.GradientFull, canvas.Cyan)
	draw.TriangleShaded(m.canvas, v0, v1, v2, gradShader)

	// Animated shader
	t := float64(m.frame) / 30.0
	animShader := func(u, v uint8, x, y int) (rune, canvas.Cell) {
		wave := math.Sin(float64(u)*0.05+t*2) * math.Cos(float64(v)*0.05+t*2)
		intensity := uint8((wave + 1) * 127)
		char := draw.GradientFull.Char(intensity)
		return char, canvas.NewCell(char, canvas.Green)
	}

	v3 := draw.NewVertex2D(45, 5, 0, 0)
	v4 := draw.NewVertex2D(35, 18, 255, 0)
	v5 := draw.NewVertex2D(55, 18, 0, 255)
	draw.TriangleShaded(m.canvas, v3, v4, v5, animShader)

	// Solid shader
	solidShader := draw.SolidShader('█', canvas.Magenta)
	v6 := draw.NewVertex2D(70, 5, 0, 0)
	v7 := draw.NewVertex2D(60, 18, 0, 0)
	v8 := draw.NewVertex2D(78, 18, 0, 0)
	draw.TriangleShaded(m.canvas, v6, v7, v8, solidShader)

	draw.PutString(m.canvas, 35, 20, "Gradient    Animated    Solid", canvas.NewCell(' ', canvas.White))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderTextures() {
	draw.PutString(m.canvas, 2, 1, "Demo 5: Textures", canvas.NewCell(' ', canvas.Yellow))

	// Create a simple texture
	tex := draw.NewTextureFromStrings([]string{
		"######",
		"#    #",
		"# ** #",
		"# ** #",
		"#    #",
		"######",
	}, canvas.White)

	// Apply texture to triangles
	texShader := draw.TextureShader(tex)

	v0 := draw.NewVertex2D(20, 5, 0, 0)
	v1 := draw.NewVertex2D(5, 18, 255, 255)
	v2 := draw.NewVertex2D(35, 18, 255, 0)
	draw.TriangleShaded(m.canvas, v0, v1, v2, texShader)

	// Checker texture
	checker := draw.CheckerTexture(8, 8,
		canvas.NewCell('█', canvas.White),
		canvas.NewCell('█', canvas.Black))
	checkerShader := draw.TextureShader(checker)

	v3 := draw.NewVertex2D(55, 5, 0, 0)
	v4 := draw.NewVertex2D(40, 18, 255, 255)
	v5 := draw.NewVertex2D(70, 18, 255, 0)
	draw.TriangleShaded(m.canvas, v3, v4, v5, checkerShader)

	draw.PutString(m.canvas, 10, 20, "Custom Texture", canvas.NewCell(' ', canvas.White))
	draw.PutString(m.canvas, 48, 20, "Checker", canvas.NewCell(' ', canvas.White))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderSubcell() {
	draw.PutString(m.canvas, 2, 1, "Demo 6: Sub-cell Rendering (Braille 2x4)", canvas.NewCell(' ', canvas.Yellow))

	if m.subcell == nil {
		return
	}

	m.subcell.Clear()

	// Draw in high resolution (2x width, 4x height)
	t := float64(m.frame) / 30.0

	// Sine wave
	for x := 0; x < m.subcell.Width(); x++ {
		y := int(float64(m.subcell.Height())/2 + math.Sin(float64(x)*0.1+t*2)*float64(m.subcell.Height())/4)
		m.subcell.SetPixel(x, y, true)
	}

	// Rotating line
	cx := m.subcell.Width() / 2
	cy := m.subcell.Height() / 2
	length := 30
	x2 := cx + int(float64(length)*math.Cos(t))
	y2 := cy + int(float64(length)*math.Sin(t))
	m.subcell.DrawLine(cx, cy, x2, y2)

	// Circle
	m.subcell.DrawCircle(cx+40, cy, 20)

	// Flush to canvas
	m.subcell.Flush(canvas.Cyan, canvas.Black)

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderEasing() {
	draw.PutString(m.canvas, 2, 1, "Demo 7: Easing & Tweens (EaseOutBounce)", canvas.NewCell(' ', canvas.Yellow))

	// Draw track
	draw.HLine(m.canvas, 5, 70, 12, canvas.NewCell('-', canvas.White))

	// Draw animated ball - get value directly from tween
	x := 5
	if m.tween != nil {
		x = int(m.tween.Value())
	}
	draw.FillCircle(m.canvas, x, 12, 2, canvas.NewCell('●', canvas.Green))

	// Show easing names
	easings := []string{
		"Linear", "EaseInQuad", "EaseOutQuad", "EaseInOutQuad",
		"EaseInCubic", "EaseOutCubic", "EaseInOutCubic",
		"EaseOutBounce", "EaseOutElastic", "EaseInOutBack",
	}
	for i, name := range easings {
		y := 15 + i/3
		x := 5 + (i%3)*25
		draw.PutString(m.canvas, x, y, name, canvas.NewCell(' ', canvas.Cyan))
	}

	progress := ""
	if m.tween != nil {
		progress = fmt.Sprintf("Progress: %.1f%%", m.tween.Progress()*100)
	}
	draw.PutString(m.canvas, 2, 20, progress, canvas.NewCell(' ', canvas.Yellow))

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderTimeline() {
	draw.PutString(m.canvas, 2, 1, "Demo 8: Timeline Animation (GSAP-style)", canvas.NewCell(' ', canvas.Yellow))

	// Draw path
	draw.Rect(m.canvas, 5, 5, 70, 18, canvas.NewCell('·', canvas.White))

	// Calculate position using frame-based timeline simulation
	// Total cycle: 4 seconds (120 frames at 30fps)
	// Phase 1: X 5->70 (0-30 frames)
	// Phase 2: Y 5->18 (30-60 frames)
	// Phase 3: X 70->5 (60-90 frames)
	// Phase 4: Y 18->5 (90-120 frames)
	cycleFrame := m.frame % 120
	phase := cycleFrame / 30
	phaseProgress := float64(cycleFrame%30) / 30.0

	// Apply easing (EaseInOutQuad)
	eased := anim.EaseInOutQuad(phaseProgress)

	var x, y float64
	switch phase {
	case 0: // X moves right
		x = 5 + eased*65
		y = 5
	case 1: // Y moves down
		x = 70
		y = 5 + eased*13
	case 2: // X moves left
		x = 70 - eased*65
		y = 18
	case 3: // Y moves up
		x = 5
		y = 18 - eased*13
	}

	// Draw animated ball
	draw.FillCircle(m.canvas, int(x), int(y), 1, canvas.NewCell('●', canvas.Magenta))

	// Draw trail
	for i := 1; i <= 5; i++ {
		trailX := int(x) - i
		if trailX > 5 && trailX < 70 {
			intensity := uint8(255 - i*40)
			char := draw.GradientMin.Char(intensity)
			m.canvas.SetCell(trailX, int(y), canvas.NewCell(char, canvas.Magenta))
		}
	}

	info := "Sequenced: X tween -> Y tween -> X tween -> Y tween (repeat)"
	draw.PutString(m.canvas, 2, 20, info, canvas.NewCell(' ', canvas.Cyan))

	totalProgress := float64(cycleFrame) / 120.0 * 100
	phaseNames := []string{"X: 5→70", "Y: 5→18", "X: 70→5", "Y: 18→5"}
	progress := fmt.Sprintf("Phase %d (%s) | Total: %.0f%%", phase+1, phaseNames[phase], totalProgress)
	draw.PutString(m.canvas, 2, 21, progress, canvas.NewCell(' ', canvas.Yellow))

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderSVG() {
	draw.PutString(m.canvas, 2, 1, "Demo 9: SVG Path Rendering", canvas.NewCell(' ', canvas.Yellow))

	// Parse a heart SVG path
	heartPath := "M 20 10 C 20 5 15 0 10 0 C 0 0 0 10 0 10 C 0 20 10 25 20 35 C 30 25 40 20 40 10 C 40 10 40 0 30 0 C 25 0 20 5 20 10 Z"
	path, err := svg.ParsePath(heartPath)
	if err != nil {
		draw.PutString(m.canvas, 2, 5, "Error parsing path", canvas.NewCell(' ', canvas.Red))
		return
	}

	// Get points and draw
	points := path.ToPoints(50)

	// Scale and offset
	offsetX := 20.0
	offsetY := 5.0
	scale := 0.4

	// Draw path with animation (progressive reveal) - 3 second cycle
	t := float64(m.frame) / 90.0 // 90 frames = 3 seconds at 30fps
	progress := math.Mod(t, 1.0)
	numPoints := int(float64(len(points)) * progress)
	if numPoints < 1 {
		numPoints = 1
	}

	// Draw completed portion
	for i := 1; i < numPoints && i < len(points); i++ {
		x1 := int(points[i-1].X*scale + offsetX)
		y1 := int(points[i-1].Y*scale*0.5 + offsetY)
		x2 := int(points[i].X*scale + offsetX)
		y2 := int(points[i].Y*scale*0.5 + offsetY)
		draw.Line(m.canvas, x1, y1, x2, y2, canvas.NewCell('♥', canvas.Red))
	}

	// Show progress indicator
	progressPct := fmt.Sprintf("Drawing: %.0f%%", progress*100)
	draw.PutString(m.canvas, 2, 18, progressPct, canvas.NewCell(' ', canvas.Red))

	// Star path
	starPath := "M 25 0 L 31 18 L 50 18 L 35 29 L 41 47 L 25 36 L 9 47 L 15 29 L 0 18 L 19 18 Z"
	star, _ := svg.ParsePath(starPath)
	starPoints := star.ToPoints(30)

	offsetX = 55.0
	for i := 1; i < len(starPoints); i++ {
		x1 := int(starPoints[i-1].X*0.3 + offsetX)
		y1 := int(starPoints[i-1].Y*0.3*0.5 + offsetY)
		x2 := int(starPoints[i].X*0.3 + offsetX)
		y2 := int(starPoints[i].Y*0.3*0.5 + offsetY)
		draw.Line(m.canvas, x1, y1, x2, y2, canvas.NewCell('★', canvas.Yellow))
	}

	// Path info
	info := fmt.Sprintf("Heart: %.0f length, %d points | Star: %.0f length",
		path.Length(), len(points), star.Length())
	draw.PutString(m.canvas, 2, 20, info, canvas.NewCell(' ', canvas.Cyan))

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) setup3D() {
	// Create camera - use aspect=1.0 since viewport transform handles terminal cell aspect
	m.camera = gl.NewPerspectiveCamera(45, 1.0, 0.1, 100)
	m.camera.Position = glmath.Vec3{X: 0, Y: 0, Z: 6}
	m.camera.Target = glmath.Vec3{X: 0, Y: 0, Z: 0}

	// Create renderer
	m.renderer = gl.NewRenderer(m.canvas, m.camera)
	m.renderer.BackfaceCull = true

	// Create meshes (scaled down to fit nicely)
	m.cubeMesh = gl.NewCube()
	m.cubeMesh.SetScale(0.8)
	m.pyramidMesh = gl.NewPyramid()
	m.pyramidMesh.SetScale(0.8)

	// Set up lighting
	m.renderer.SetDirectionalLight(&gl.DirectionalLight{
		Direction: glmath.Vec3{X: -1, Y: -1, Z: -1}.Normalize(),
		Intensity: 0.8,
	})
	m.renderer.SetAmbientLight(&gl.AmbientLight{
		Intensity: 0.2,
	})
}

func (m *model) renderKitty() {
	draw.PutString(m.canvas, 2, 1, "Demo 0: Kitty Image Protocol", canvas.NewCell(' ', canvas.Yellow))

	if m.kitty == nil {
		draw.PutString(m.canvas, 2, 5, "Kitty backend not initialized", canvas.NewCell(' ', canvas.Red))
		return
	}

	// Check if Kitty is supported
	if !canvas.IsKittySupported() {
		draw.PutString(m.canvas, 2, 5, "Kitty protocol not detected.", canvas.NewCell(' ', canvas.Red))
		draw.PutString(m.canvas, 2, 6, "Supported terminals: Kitty, WezTerm, Ghostty", canvas.NewCell(' ', canvas.White))
		draw.PutString(m.canvas, 2, 8, "Rendering preview to canvas instead:", canvas.NewCell(' ', canvas.Yellow))
	}

	// Clear and draw to Kitty backend
	m.kitty.Clear()

	// Animated graphics
	t := float64(m.frame) / 30.0

	// Draw gradient background
	for y := 0; y < m.kitty.Height(); y++ {
		for x := 0; x < m.kitty.Width(); x++ {
			// Create a plasma effect
			v1 := math.Sin(float64(x)*0.05 + t)
			v2 := math.Sin(float64(y)*0.05 + t*0.7)
			v3 := math.Sin((float64(x)+float64(y))*0.05 + t*0.5)
			v := (v1 + v2 + v3 + 3) / 6

			r := uint8(v * 100)
			g := uint8((1 - v) * 150)
			b := uint8(128 + v*127)
			m.kitty.SetPixel(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Draw rotating shape
	cx := m.kitty.Width() / 2
	cy := m.kitty.Height() / 2
	radius := 30

	// Draw filled circle with gradient
	for angle := 0.0; angle < 2*math.Pi; angle += 0.1 {
		for r := 0; r < radius; r++ {
			x := cx + int(float64(r)*math.Cos(angle+t))
			y := cy + int(float64(r)*math.Sin(angle+t))
			intensity := uint8(255 - r*255/radius)
			m.kitty.SetPixel(x, y, color.RGBA{R: intensity, G: 200, B: 255 - intensity, A: 255})
		}
	}

	// Draw spinning lines
	for i := 0; i < 6; i++ {
		angle := t + float64(i)*math.Pi/3
		x2 := cx + int(float64(radius+20)*math.Cos(angle))
		y2 := cy + int(float64(radius+20)*math.Sin(angle))
		m.kitty.DrawLine(cx, cy, x2, y2, color.RGBA{R: 255, G: 255, B: 0, A: 255})
	}

	// Show preview on canvas (approximate)
	draw.PutString(m.canvas, 2, 10, "Kitty renders true pixels (160x96)", canvas.NewCell(' ', canvas.Cyan))
	draw.PutString(m.canvas, 2, 11, "Canvas shows character approximation:", canvas.NewCell(' ', canvas.White))

	// Approximate Kitty output on canvas
	scaleX := m.kitty.Width() / (width - 4)
	scaleY := m.kitty.Height() / (height - 15)
	for cy := 0; cy < height-15; cy++ {
		for cx := 0; cx < width-4; cx++ {
			px := cx * scaleX
			py := cy * scaleY
			if px < m.kitty.Width() && py < m.kitty.Height() {
				c := m.kitty.GetPixel(px, py)
				intensity := (int(c.R) + int(c.G) + int(c.B)) / 3
				char := draw.GradientFull.Char(uint8(intensity))
				m.canvas.SetCell(2+cx, 13+cy, canvas.NewCell(char, canvas.RGB(c.R, c.G, c.B)))
			}
		}
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) render3DCube() {
	draw.PutString(m.canvas, 2, 1, "Demo A: 3D Rotating Cube (Shaded) [New Pipeline]", canvas.NewCell(' ', canvas.Yellow))

	if m.renderer == nil {
		return
	}

	m.renderer.Clear()
	m.renderer.ShadingMode = gl.ShadingFlat

	// Rotate cube
	t := float64(m.frame) / 30.0
	m.cubeMesh.SetRotation(t*30, t*45, t*20)

	// Render using new shader pipeline with lighting
	m.renderer.RenderMeshLit(m.cubeMesh)

	info := fmt.Sprintf("Rotation: (%.0f, %.0f, %.0f) degrees", t*30, t*45, t*20)
	draw.PutString(m.canvas, 2, height-2, info, canvas.NewCell(' ', canvas.Cyan))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) render3DWireframe() {
	draw.PutString(m.canvas, 2, 1, "Demo B: 3D Wireframe Rendering [New Pipeline]", canvas.NewCell(' ', canvas.Yellow))

	if m.renderer == nil {
		return
	}

	m.renderer.Clear()
	m.renderer.BackfaceCull = false // Show all edges in wireframe

	// Rotate cube
	t := float64(m.frame) / 30.0
	m.cubeMesh.SetRotation(t*20, t*35, 0)
	m.cubeMesh.SetPosition(-1.5, 0, 0)

	// Rotate pyramid
	m.pyramidMesh.SetRotation(t*25, t*40, t*15)
	m.pyramidMesh.SetPosition(1.5, 0, 0)

	// Render both using new shader pipeline with wireframe
	m.renderer.RenderMeshWireframe(m.cubeMesh, '#', canvas.Cyan)
	m.renderer.RenderMeshWireframe(m.pyramidMesh, '#', canvas.Yellow)

	draw.PutString(m.canvas, 2, height-2, "Cube (left) and Pyramid (right)", canvas.NewCell(' ', canvas.Cyan))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) render3DShading() {
	draw.PutString(m.canvas, 2, 1, "Demo C: 3D Shading Modes [New Pipeline]", canvas.NewCell(' ', canvas.Yellow))

	if m.renderer == nil {
		return
	}

	m.renderer.Clear()

	// Cycle through render modes based on time
	t := float64(m.frame) / 30.0
	modeIndex := (m.frame / 90) % 4 // Change mode every 3 seconds (4 modes now)

	modeNames := []string{"Shaded", "Wireframe", "Solid", "Outlined"}

	m.renderer.ShadingMode = gl.ShadingFlat
	m.renderer.BackfaceCull = true

	// Rotate cube
	m.cubeMesh.SetRotation(t*30, t*45, t*20)
	m.cubeMesh.SetPosition(0, 0, 0)

	// Render using appropriate new pipeline method based on mode
	switch modeIndex {
	case 0: // Shaded
		m.renderer.RenderMeshLit(m.cubeMesh)
	case 1: // Wireframe
		m.renderer.RenderMeshWireframe(m.cubeMesh, '#', canvas.White)
	case 2: // Solid
		m.renderer.RenderMeshSolid(m.cubeMesh, '#', canvas.Green)
	case 3: // Outlined (solid + wireframe)
		m.renderer.RenderMeshSolid(m.cubeMesh, '.', canvas.Blue)
		m.renderer.RenderMeshWireframe(m.cubeMesh, '#', canvas.White)
	}

	// Show current mode
	info := fmt.Sprintf("Mode: %s (cycles every 3s)", modeNames[modeIndex])
	draw.PutString(m.canvas, 2, height-3, info, canvas.NewCell(' ', canvas.Yellow))

	draw.PutString(m.canvas, 2, height-2, "Modes: Shaded, Wireframe, Solid, Outlined", canvas.NewCell(' ', canvas.Cyan))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

// ============================================================================
// TermGL-C-Plus Demo Ports (z,x,c,v,b,n,m keys)
// ============================================================================

func (m *model) setupTeapot() {
	// Setup camera
	m.camera = gl.NewPerspectiveCamera(45, 1.0, 0.1, 100)
	m.camera.Position = glmath.Vec3{X: 0, Y: 0, Z: 3}
	m.camera.Target = glmath.Vec3{X: 0, Y: 0, Z: 0}

	// Create renderer
	m.renderer = gl.NewRenderer(m.canvas, m.camera)
	m.renderer.BackfaceCull = true

	// Try to load teapot STL
	teapot, err := gl.LoadSTLFile("ref/TermGL-C-Plus/utah_teapot.stl")
	if err != nil {
		// Fallback to a simple cube if STL not found
		m.teapotMesh = gl.NewCube()
		m.teapotMesh.SetScale(0.5)
	} else {
		m.teapotMesh = teapot
		m.teapotMesh.SetScale(0.015) // STL is large, scale down
		m.teapotMesh.ComputeVertexNormals()
	}

	// Set up lighting
	m.renderer.SetDirectionalLight(&gl.DirectionalLight{
		Direction: glmath.Vec3{X: -0.5, Y: -1, Z: -0.5}.Normalize(),
		Intensity: 0.9,
	})
	m.renderer.SetAmbientLight(&gl.AmbientLight{
		Intensity: 0.1,
	})
}

func (m *model) renderTeapot() {
	draw.PutString(m.canvas, 2, 1, "Demo Z: Utah Teapot (STL + Lighting) [New Pipeline]", canvas.NewCell(' ', canvas.Yellow))

	if m.renderer == nil || m.teapotMesh == nil {
		draw.PutString(m.canvas, 2, 5, "Loading teapot...", canvas.NewCell(' ', canvas.White))
		return
	}

	m.renderer.Clear()
	m.renderer.ShadingMode = gl.ShadingSmooth

	// Set custom shade ramp with colors like TermGL-C-Plus
	// Uses numbers 1-8 with different colors for each shade level
	m.renderer.SetShadingRamp(" 12345678#", []canvas.Color{
		canvas.Black,              // ' ' - darkest
		canvas.RGB(128, 0, 128),   // '1' - purple
		canvas.RGB(0, 0, 255),     // '2' - blue
		canvas.RGB(0, 128, 128),   // '3' - teal
		canvas.RGB(0, 255, 0),     // '4' - green
		canvas.RGB(255, 255, 0),   // '5' - yellow
		canvas.RGB(255, 128, 0),   // '6' - orange
		canvas.RGB(255, 0, 0),     // '7' - red
		canvas.RGB(255, 128, 128), // '8' - pink
		canvas.RGB(255, 255, 255), // '#' - white (brightest)
	})

	// Rotate teapot around Z axis (like C version)
	t := float64(m.frame) / 30.0
	m.teapotMesh.SetRotation(-90, t*30, 0) // -90 to stand upright

	// Render using new shader pipeline with lighting
	m.renderer.RenderMeshLit(m.teapotMesh)

	info := fmt.Sprintf("Triangles: %d | Rotation: %.0f°", m.teapotMesh.TriangleCount(), t*30)
	draw.PutString(m.canvas, 2, height-2, info, canvas.NewCell(' ', canvas.Cyan))
	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderColorPalette() {
	draw.PutString(m.canvas, 2, 1, "Demo X: Color Palette (TermGL-C-Plus Style)", canvas.NewCell(' ', canvas.Yellow))

	// Color names matching TermGL-C-Plus
	colors := []struct {
		name  string
		color canvas.Color
		char  rune
	}{
		{"BLACK", canvas.Black, 'K'},
		{"RED", canvas.Red, 'R'},
		{"GREEN", canvas.Green, 'G'},
		{"YELLOW", canvas.Yellow, 'Y'},
		{"BLUE", canvas.Blue, 'B'},
		{"PURPLE", canvas.Magenta, 'P'},
		{"CYAN", canvas.Cyan, 'C'},
		{"WHITE", canvas.White, 'W'},
	}

	// Headers
	draw.PutString(m.canvas, 2, 3, "Base Colors:", canvas.NewCell(' ', canvas.White))
	draw.PutString(m.canvas, 2, 4, "K  R  G  Y  B  P  C  W", canvas.NewCell(' ', canvas.White))

	// Row 1: Foreground colors on black background
	draw.PutString(m.canvas, 2, 6, "FG on Black:", canvas.NewCell(' ', canvas.White))
	for i, c := range colors {
		m.canvas.SetCell(14+i*3, 6, canvas.NewCell(c.char, c.color))
	}

	// Row 2: Black text on colored backgrounds
	draw.PutString(m.canvas, 2, 8, "BG Colors:", canvas.NewCell(' ', canvas.White))
	for i, c := range colors {
		cell := canvas.Cell{Rune: ' ', Foreground: canvas.Black, Background: c.color}
		m.canvas.SetCell(14+i*3, 8, cell)
		m.canvas.SetCell(15+i*3, 8, cell)
	}

	// Show bright/high intensity colors (ANSI 8-15)
	draw.PutString(m.canvas, 2, 10, "High Intensity (Bright):", canvas.NewCell(' ', canvas.White))
	brightColors := []canvas.Color{
		canvas.BrightBlack, canvas.BrightRed, canvas.BrightGreen, canvas.BrightYellow,
		canvas.BrightBlue, canvas.BrightMagenta, canvas.BrightCyan, canvas.BrightWhite,
	}
	for i, c := range brightColors {
		m.canvas.SetCell(2+i*3, 11, canvas.NewCell(colors[i].char, c))
	}

	// Show RGB gradient
	draw.PutString(m.canvas, 2, 13, "24-bit RGB Gradient:", canvas.NewCell(' ', canvas.White))
	for i := 0; i < 60; i++ {
		r := uint8(i * 255 / 60)
		g := uint8(0)
		b := uint8(255 - i*255/60)
		m.canvas.SetCell(2+i, 14, canvas.NewCell('█', canvas.RGB(r, g, b)))
	}

	// Show grayscale
	draw.PutString(m.canvas, 2, 16, "Grayscale:", canvas.NewCell(' ', canvas.White))
	for i := 0; i < 40; i++ {
		v := uint8(i * 255 / 40)
		m.canvas.SetCell(12+i, 16, canvas.NewCell('█', canvas.RGB(v, v, v)))
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderMandelbrot() {
	draw.PutString(m.canvas, 2, 0, "Demo C: Mandelbrot Fractal", canvas.NewCell(' ', canvas.Yellow))

	// Animated zoom parameters (like C version)
	// Zooms from (-1.0, 0) toward (-1.31, 0)
	zoomProgress := float64(m.frame%180) / 180.0 // 6 second cycle

	// Interpolate center and zoom
	startCenterX := -1.0
	endCenterX := -1.31
	centerX := startCenterX + (endCenterX-startCenterX)*zoomProgress
	centerY := 0.0

	startWidth := 2.5
	endWidth := 0.12
	viewWidth := startWidth - (startWidth-endWidth)*zoomProgress
	viewHeight := viewWidth * float64(height-2) / float64(width) * 2.0 // Aspect ratio

	// Render Mandelbrot
	maxIter := 255
	for py := 0; py < height-2; py++ {
		for px := 0; px < width; px++ {
			// Map pixel to complex plane
			x0 := centerX - viewWidth/2 + float64(px)/float64(width)*viewWidth
			y0 := centerY - viewHeight/2 + float64(py)/float64(height-2)*viewHeight

			// Mandelbrot iteration
			x, y := 0.0, 0.0
			iter := 0
			for x*x+y*y <= 4 && iter < maxIter {
				xTemp := x*x - y*y + x0
				y = 2*x*y + y0
				x = xTemp
				iter++
			}

			// Map iteration count to gradient character
			if iter == maxIter {
				m.canvas.SetCell(px, py+1, canvas.NewCell(' ', canvas.Black))
			} else {
				intensity := uint8(iter * 255 / maxIter)
				char := draw.GradientFull.Char(intensity)
				m.canvas.SetCell(px, py+1, canvas.NewCell(char, canvas.White))
			}
		}
	}

	info := fmt.Sprintf("Zoom: %.2fx | Center: (%.4f, %.4f)", startWidth/viewWidth, centerX, centerY)
	draw.PutString(m.canvas, 2, height-1, info, canvas.NewCell(' ', canvas.Cyan))
}

func (m *model) renderKeyboardDemo() {
	draw.PutString(m.canvas, 2, 1, "Demo V: Keyboard Input (TermGL-C-Plus Style)", canvas.NewCell(' ', canvas.Yellow))

	draw.PutString(m.canvas, 2, 5, "Press any key to see it displayed below.", canvas.NewCell(' ', canvas.White))
	draw.PutString(m.canvas, 2, 6, "(Press Q or Esc to exit)", canvas.NewCell(' ', canvas.Cyan))

	// Display last pressed key
	keyDisplay := "NONE"
	if m.lastKey != "" {
		keyDisplay = m.lastKey
	}

	label := fmt.Sprintf("Pressed key: %s", keyDisplay)
	draw.PutString(m.canvas, 2, 10, label, canvas.NewCell(' ', canvas.Green))

	// Draw a visual representation of the key
	if m.lastKey != "" {
		keyLen := len(m.lastKey)
		boxWidth := keyLen + 4
		if boxWidth < 8 {
			boxWidth = 8
		}
		startX := 2
		startY := 12

		// Draw key box
		draw.Rect(m.canvas, startX, startY, startX+boxWidth, startY+4, canvas.NewCell('─', canvas.White))
		draw.PutStringCentered(m.canvas, startY+2, m.lastKey, canvas.NewCell(' ', canvas.BrightYellow))
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back  [Q] Quit", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderTexturedCube() {
	draw.PutString(m.canvas, 2, 1, "Demo B: Textured Cube (UV Mapping)", canvas.NewCell(' ', canvas.Yellow))

	if m.renderer == nil {
		return
	}

	// Create texture on first render (matching C version's 6x6 texture)
	if m.cubeTexture == nil {
		// Texture pattern matching TermGL-C-Plus demo_texture_data.c
		texChars := []string{
			"######",
			"# 1 2#",
			"#3 4 #",
			"# 5 6#",
			"#7 8 #",
			"######",
		}
		// Color mapping for each character (matching C version)
		colors := map[rune]canvas.Color{
			'#': canvas.White,
			' ': canvas.Black,
			'1': canvas.Red,
			'2': canvas.Green,
			'3': canvas.Yellow,
			'4': canvas.Blue,
			'5': canvas.Magenta,
			'6': canvas.Cyan,
			'7': canvas.BrightRed,
			'8': canvas.BrightGreen,
		}
		m.cubeTexture = gl.NewTextureFromCharsAndColors(texChars, colors)
	}

	m.renderer.Clear()

	// Rotate cube around two axes (like C version)
	t := float64(m.frame) / 30.0
	m.cubeMesh.SetRotation(t*20, t*30, 0)
	m.cubeMesh.SetPosition(0, 0, 0)

	// Render cube with texture using the shader pipeline
	m.renderer.RenderMeshTextured(m.cubeMesh, m.cubeTexture)

	// Draw texture preview in corner
	draw.PutString(m.canvas, 2, height-6, "Texture Preview:", canvas.NewCell(' ', canvas.White))
	texChars := []string{
		"######",
		"# 1 2#",
		"#3 4 #",
		"# 5 6#",
		"#7 8 #",
		"######",
	}
	colors := map[rune]canvas.Color{
		'#': canvas.White,
		' ': canvas.Black,
		'1': canvas.Red,
		'2': canvas.Green,
		'3': canvas.Yellow,
		'4': canvas.Blue,
		'5': canvas.Magenta,
		'6': canvas.Cyan,
		'7': canvas.BrightRed,
		'8': canvas.BrightGreen,
	}
	for y, row := range texChars {
		for x, ch := range row {
			color := colors[ch]
			if color == "" {
				color = canvas.White
			}
			m.canvas.SetCell(20+x, height-6+y, canvas.NewCell(ch, color))
		}
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderRGBCircles() {
	draw.PutString(m.canvas, 2, 1, "Demo N: RGB Circles (24-bit Color)", canvas.NewCell(' ', canvas.Yellow))

	// Three overlapping circles like TermGL-C-Plus
	// Red centered at (120, 100), Green at (100, 140), Blue at (140, 140)
	// Scaled to fit terminal

	// Scale factors for terminal
	scaleX := float64(width) / 200.0
	scaleY := float64(height-2) / 180.0

	// Circle centers (scaled)
	redCX, redCY := 60.0*scaleX, 50.0*scaleY
	greenCX, greenCY := 50.0*scaleX, 70.0*scaleY
	blueCX, blueCY := 70.0*scaleX, 70.0*scaleY

	for py := 1; py < height-1; py++ {
		for px := 0; px < width; px++ {
			x := float64(px)
			y := float64(py)

			// Calculate distance to each circle center
			// Using elliptical distance to account for terminal aspect ratio
			redDist := math.Sqrt((x-redCX)*(x-redCX)*0.25 + (y-redCY)*(y-redCY))
			greenDist := math.Sqrt((x-greenCX)*(x-greenCX)*0.25 + (y-greenCY)*(y-greenCY))
			blueDist := math.Sqrt((x-blueCX)*(x-blueCX)*0.25 + (y-blueCY)*(y-blueCY))

			// Map distance to intensity (closer = brighter)
			radius := 12.0
			redIntensity := math.Max(0, 1-redDist/radius)
			greenIntensity := math.Max(0, 1-greenDist/radius)
			blueIntensity := math.Max(0, 1-blueDist/radius)

			// Combine RGB values
			r := uint8(redIntensity * 255)
			g := uint8(greenIntensity * 255)
			b := uint8(blueIntensity * 255)

			if r > 0 || g > 0 || b > 0 {
				m.canvas.SetCell(px, py, canvas.NewCell('█', canvas.RGB(r, g, b)))
			}
		}
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back", canvas.NewCell(' ', canvas.White))
}

func (m *model) renderMouseDemo() {
	draw.PutString(m.canvas, 2, 1, "Demo M: Mouse Tracking (TermGL-C-Plus Style)", canvas.NewCell(' ', canvas.Yellow))

	draw.PutString(m.canvas, 2, 4, "Move the mouse around the terminal.", canvas.NewCell(' ', canvas.White))

	// Display mouse position
	posStr := fmt.Sprintf("Mouse position: X=%d, Y=%d", m.mouseX, m.mouseY)
	draw.PutString(m.canvas, 2, 7, posStr, canvas.NewCell(' ', canvas.Green))

	// Display last button action
	buttonStr := fmt.Sprintf("Latest action: %s", m.mouseButton)
	draw.PutString(m.canvas, 2, 9, buttonStr, canvas.NewCell(' ', canvas.Cyan))

	// Draw crosshair at mouse position
	if m.mouseX >= 0 && m.mouseX < width && m.mouseY >= 0 && m.mouseY < height {
		// Horizontal line
		for x := 0; x < width; x++ {
			if x != m.mouseX {
				cell := m.canvas.GetCell(x, m.mouseY)
				if cell.Rune == 0 || cell.Rune == ' ' {
					m.canvas.SetCell(x, m.mouseY, canvas.NewCell('─', canvas.BrightBlack))
				}
			}
		}
		// Vertical line
		for y := 3; y < height-1; y++ {
			if y != m.mouseY {
				cell := m.canvas.GetCell(m.mouseX, y)
				if cell.Rune == 0 || cell.Rune == ' ' {
					m.canvas.SetCell(m.mouseX, y, canvas.NewCell('│', canvas.BrightBlack))
				}
			}
		}
		// Crosshair center
		m.canvas.SetCell(m.mouseX, m.mouseY, canvas.NewCell('┼', canvas.BrightYellow))
	}

	draw.PutString(m.canvas, 2, height-1, "[Esc] Back  (Mouse tracking enabled)", canvas.NewCell(' ', canvas.White))
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
