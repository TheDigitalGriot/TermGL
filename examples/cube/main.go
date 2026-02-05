// Spinning cube example demonstrating TermGL rendering.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl/anim"
	"github.com/charmbracelet/termgl/canvas"
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
	canvas   *canvas.Canvas
	renderer *gl.Renderer
	camera   *gl.Camera
	cube     *gl.Mesh

	rotX *anim.AnimatedFloat
	rotY *anim.AnimatedFloat

	width  int
	height int

	autoRotate bool
	angle      float64
}

func initialModel() model {
	width := 80
	height := 24

	// Create canvas
	c := canvas.New(width, height)

	// Create camera
	cam := gl.NewPerspectiveCamera(30, c.Aspect(), 0.1, 1000)
	cam.SetPosition(0, 0, 8)
	cam.LookAt(math.Vec3{})

	// Create renderer
	r := gl.NewRenderer(c, cam)
	r.RenderMode = gl.RenderShaded
	r.ShadingMode = gl.ShadingFlat
	r.SetDirectionalLight(gl.NewDirectionalLight(math.Vec3{X: 0.5, Y: 0.5, Z: -1}, 1.0))
	r.SetAmbientLight(gl.NewAmbientLight(0.2))

	// Create cube
	cube := gl.NewCube()

	// Create animated rotation
	rotX := anim.NewAnimatedFloat(0, frequency, damping, fps)
	rotY := anim.NewAnimatedFloat(0, frequency, damping, fps)

	return model{
		canvas:     c,
		renderer:   r,
		camera:     cam,
		cube:       cube,
		rotX:       rotX,
		rotY:       rotY,
		width:      width,
		height:     height,
		autoRotate: true,
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
		case "1":
			m.renderer.RenderMode = gl.RenderShaded
			m.renderer.ShadingMode = gl.ShadingFlat
		case "2":
			m.renderer.RenderMode = gl.RenderShaded
			m.renderer.ShadingMode = gl.ShadingSmooth
		case "3":
			m.renderer.RenderMode = gl.RenderWireframe
		case "4":
			m.renderer.RenderMode = gl.RenderSolid
		case "5":
			m.renderer.RenderMode = gl.RenderOutlined
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.canvas.Resize(msg.Width, msg.Height)
		m.camera.SetAspect(m.canvas.Aspect())
		return m, nil

	case tickMsg:
		// Auto-rotate if enabled
		if m.autoRotate {
			m.angle += 1.5
			m.rotY.Set(m.angle)
			m.rotX.Set(m.angle * 0.3)
		}

		// Update spring animations
		m.rotX.Update()
		m.rotY.Update()

		// Apply rotation to cube
		m.cube.SetRotation(m.rotX.Get(), m.rotY.Get(), 0)

		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	// Clear and render
	m.renderer.Clear()
	m.renderer.RenderMesh(m.cube)

	// Add help text at bottom
	help := "\n[arrows] rotate  [space] toggle auto  [1-5] render mode  [q] quit"

	return m.canvas.String() + help
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
