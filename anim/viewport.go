package anim

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl/canvas"
	"github.com/charmbracelet/termgl/gl"
)

// TickMsg is sent on each animation frame.
type TickMsg time.Time

// Viewport is a Bubble Tea Model that renders a 3D scene.
type Viewport struct {
	Canvas   *canvas.Canvas
	Renderer *gl.Renderer
	Camera   *gl.Camera
	Meshes   []*gl.Mesh

	// Settings
	FPS     int
	Width   int
	Height  int
	Running bool

	// Callbacks
	OnUpdate func(dt float64) // Called each frame before rendering

	// Internal
	lastTick time.Time
}

// ViewportOption configures a viewport.
type ViewportOption func(*Viewport)

// WithFPS sets the target frame rate.
func WithFPS(fps int) ViewportOption {
	return func(v *Viewport) {
		v.FPS = fps
	}
}

// WithSize sets the viewport size in cells.
func WithSize(width, height int) ViewportOption {
	return func(v *Viewport) {
		v.Width = width
		v.Height = height
	}
}

// New creates a new viewport with the given options.
func New(opts ...ViewportOption) *Viewport {
	v := &Viewport{
		FPS:     30,
		Width:   80,
		Height:  24,
		Running: true,
		Meshes:  make([]*gl.Mesh, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(v)
	}

	// Create canvas and camera
	v.Canvas = canvas.New(v.Width, v.Height)
	v.Camera = gl.NewPerspectiveCamera(30, v.Canvas.Aspect(), 0.1, 1000)
	v.Camera.SetPosition(0, 0, 5)
	v.Camera.LookAt(gl.Vec3{})

	// Create renderer
	v.Renderer = gl.NewRenderer(v.Canvas, v.Camera)

	return v
}

// AddMesh adds a mesh to the scene.
func (v *Viewport) AddMesh(m *gl.Mesh) {
	v.Meshes = append(v.Meshes, m)
}

// RemoveMesh removes a mesh from the scene.
func (v *Viewport) RemoveMesh(m *gl.Mesh) {
	for i, mesh := range v.Meshes {
		if mesh == m {
			v.Meshes = append(v.Meshes[:i], v.Meshes[i+1:]...)
			return
		}
	}
}

// ClearMeshes removes all meshes.
func (v *Viewport) ClearMeshes() {
	v.Meshes = v.Meshes[:0]
}

// Resize changes the viewport dimensions.
func (v *Viewport) Resize(width, height int) {
	v.Width = width
	v.Height = height
	v.Canvas.Resize(width, height)
	v.Camera.SetAspect(v.Canvas.Aspect())
}

// tick returns a command that sends a TickMsg after the frame interval.
func (v *Viewport) tick() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(v.FPS), func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Init initializes the viewport and starts the render loop.
func (v *Viewport) Init() tea.Cmd {
	v.lastTick = time.Now()
	return v.tick()
}

// Update handles messages.
func (v *Viewport) Update(msg tea.Msg) (*Viewport, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		if !v.Running {
			return v, nil
		}

		// Calculate delta time
		now := time.Time(msg)
		dt := now.Sub(v.lastTick).Seconds()
		v.lastTick = now

		// Call update callback if set
		if v.OnUpdate != nil {
			v.OnUpdate(dt)
		}

		// Continue ticking
		return v, v.tick()

	case tea.WindowSizeMsg:
		// Resize viewport to fit terminal
		v.Resize(msg.Width, msg.Height)
		return v, nil
	}

	return v, nil
}

// View renders the viewport to a string.
func (v *Viewport) View() string {
	// Clear and render all meshes
	v.Renderer.Clear()
	for _, mesh := range v.Meshes {
		v.Renderer.RenderMesh(mesh)
	}
	return v.Canvas.String()
}

// Convenience type alias
type Vec3 = gl.Vec3
