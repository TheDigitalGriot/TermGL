// Suzanne model viewer example demonstrating OBJ loading.
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
	mesh     *gl.Mesh

	rotX *anim.AnimatedFloat
	rotY *anim.AnimatedFloat

	width  int
	height int

	autoRotate bool
	angle      float64
	err        error
}

func initialModel() model {
	width := 100
	height := 40

	// Create canvas
	c := canvas.New(width, height)

	// Create camera
	cam := gl.NewPerspectiveCamera(30, c.Aspect(), 0.1, 1000)
	cam.SetPosition(0, 0, 5)
	cam.LookAt(math.Vec3{})

	// Create renderer
	r := gl.NewRenderer(c, cam)
	r.RenderMode = gl.RenderShaded
	r.ShadingMode = gl.ShadingSmooth
	r.SetDirectionalLight(gl.NewDirectionalLight(math.Vec3{X: 0.3, Y: 0.5, Z: -1}, 1.0))
	r.SetAmbientLight(gl.NewAmbientLight(0.15))

	// Load Suzanne model
	mesh, err := gl.LoadOBJFile("models/suzanne_blender_monkey.obj")

	m := model{
		canvas:     c,
		renderer:   r,
		camera:     cam,
		mesh:       mesh,
		rotX:       anim.NewAnimatedFloat(0, frequency, damping, fps),
		rotY:       anim.NewAnimatedFloat(0, frequency, damping, fps),
		width:      width,
		height:     height,
		autoRotate: true,
		err:        err,
	}

	return m
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
			m.rotY.Set(m.rotY.Target() - 30)
			m.autoRotate = false
		case "right":
			m.rotY.Set(m.rotY.Target() + 30)
			m.autoRotate = false
		case "up":
			m.rotX.Set(m.rotX.Target() - 30)
			m.autoRotate = false
		case "down":
			m.rotX.Set(m.rotX.Target() + 30)
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
		case "+", "=":
			// Zoom in
			pos := m.camera.Position
			m.camera.SetPosition(pos.X, pos.Y, pos.Z-0.5)
		case "-":
			// Zoom out
			pos := m.camera.Position
			m.camera.SetPosition(pos.X, pos.Y, pos.Z+0.5)
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
			m.angle += 1.0
			m.rotY.Set(m.angle)
		}

		// Update spring animations
		m.rotX.Update()
		m.rotY.Update()

		// Apply rotation to mesh
		if m.mesh != nil {
			m.mesh.SetRotation(m.rotX.Get(), m.rotY.Get(), 0)
		}

		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading model: %v\n\nMake sure models/suzanne_blender_monkey.obj exists.\n\nPress q to quit.", m.err)
	}

	// Clear and render
	m.renderer.Clear()
	if m.mesh != nil {
		m.renderer.RenderMesh(m.mesh)
	}

	// Add info
	triCount := 0
	if m.mesh != nil {
		triCount = m.mesh.TriangleCount()
	}
	info := fmt.Sprintf("\nSuzanne (%d triangles) | [arrows] rotate  [+/-] zoom  [space] auto  [1-5] mode  [q] quit", triCount)

	return m.canvas.String() + info
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
