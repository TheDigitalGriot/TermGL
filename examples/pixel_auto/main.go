// Example: Auto-detecting 3D Viewer
//
// Loads Suzanne via FauxGL, auto-detects the best encoder (Sixel or ANSI),
// and renders with keyboard controls. Shows detected capabilities and active
// tier in a status line.
//
// Run with: go run ./examples/pixel_auto
//
// Controls:
//   q / Ctrl+C — quit
//   1-4 — switch blitter (ANSI mode): half-block, quadrant, sextant, braille
//   s — force Sixel mode
//   a — force ANSI mode
//   d — toggle delta encoding
//   f — toggle frequency splitting
//   e — toggle edge-aware selection
//
// Architecture doc Section 11
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl"
	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
	"github.com/charmbracelet/termgl/tier2"
	"github.com/fogleman/fauxgl"
)

type model struct {
	viewport *render.Model
	caps     render.TerminalCaps
	tier     render.OutputTier
	angle    float64
	meshNode *render.Node

	// Status display
	blitterName string
	features    string
	width       int
	height      int
	frameCount  int
}

func main() {
	// Load mesh
	mesh, err := render.LoadOBJ("models/suzanne_blender_monkey.obj")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mesh: %v\nTrying alternate path...\n", err)
		// Try from project root
		mesh, err = render.LoadOBJ("../../models/suzanne_blender_monkey.obj")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mesh: %v\nUsing cube instead.\n", err)
			mesh = render.NewCubeMesh()
		}
	}
	mesh.BiUnitCube()

	// Detect capabilities
	caps := render.Detect()
	tier := render.SelectTier(caps)

	// Build scene
	scene := render.NewScene()
	meshNode := render.NewMeshNode(mesh)
	scene.Root.AddChild(meshNode)

	scene.Camera.SetPosition(0, 0, 3)
	scene.Camera.SetTarget(0, 0, 0)

	light := render.NewDirectionalLight(0.5, 0.5, 1, fauxgl.Color{R: 1, G: 1, B: 1, A: 1}, 0.8)
	scene.Lights = append(scene.Lights, light)
	scene.Ambient = fauxgl.Color{R: 0.12, G: 0.12, B: 0.18, A: 1}

	// Select encoder based on detected tier
	var encoder render.Encoder
	var blitterName string
	var features string

	switch tier {
	case render.TierSixel:
		encoder = termgl.NewSixelEncoder(
			termgl.SixelColors(256),
			termgl.SixelDither(tier1.DitherOrdered4x4),
		)
		blitterName = "Sixel (pixel)"
		features = "RLE:on Stable:on"
	default:
		encoder = termgl.NewANSIEncoder(
			termgl.WithBlitter(tier2.SextantBlitter{}),
			termgl.WithFrequencySplit(true),
			termgl.WithDither(tier1.DitherOrdered4x4),
			termgl.WithDelta(true),
			termgl.WithEdgeAware(true),
		)
		blitterName = "Sextant (2x3)"
		features = "FreqSplit:on EdgeAware:on Delta:on Dither:Bayer4x4"
	}

	// Reserve 2 rows for status
	statusCaps := caps
	statusCaps.Height = caps.Height - 2

	viewport := render.NewModel(scene, encoder, statusCaps, 30)

	m := &model{
		viewport:    viewport,
		caps:        caps,
		tier:        tier,
		meshNode:    meshNode,
		blitterName: blitterName,
		features:    features,
		width:       caps.Width,
		height:      caps.Height,
	}

	// Set rotation callback
	viewport.OnUpdate = func(dt float64) {
		m.angle += dt * 1.0
		m.meshNode.SetRotation(0.2, m.angle, 0)
		m.frameCount++
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func (m *model) Init() tea.Cmd {
	return m.viewport.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust viewport height for status bar
		msg.Height -= 2
	}

	// Forward to viewport
	newViewport, cmd := m.viewport.Update(msg)
	m.viewport = newViewport.(*render.Model)
	return m, cmd
}

func (m *model) View() string {
	var b strings.Builder

	// Render viewport
	b.WriteString(m.viewport.View())

	// Status line 1: Tier and capabilities
	b.WriteString(fmt.Sprintf("\n Tier: %s | %s | Terminal: %s | %dx%d",
		m.tier.String(),
		m.blitterName,
		m.caps.Terminal,
		m.width,
		m.height,
	))

	// Status line 2: Features and frame count
	b.WriteString(fmt.Sprintf("\n %s | Frame: %d | Sixel:%v TrueColor:%v Unicode:%d",
		m.features,
		m.frameCount,
		m.caps.Sixel,
		m.caps.TrueColor,
		m.caps.Unicode,
	))

	return b.String()
}
