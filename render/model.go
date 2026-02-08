package render

import (
	"image"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg is the internal tick message for frame timing.
type tickMsg time.Time

// Model is the Bubble Tea model for a TermGL 3D viewport.
// It drives the render loop: tick → scene update → rasterize → encode → view.
// Architecture doc Section 6.1
type Model struct {
	Scene    *Scene
	renderer *Renderer
	encoder  Encoder
	caps     TerminalCaps
	tier     OutputTier

	// Terminal dimensions in cells
	width, height int

	// Frame timing
	fps      int
	lastTick time.Time

	// Render state
	lastFrame *image.NRGBA
	lastAux   *AuxBuffers
	output    string

	// User callbacks
	OnUpdate func(dt float64)     // Called each frame before rendering
	OnKey    func(msg tea.KeyMsg) // Called on key press

	// Whether to use aux buffers (for edge-aware ANSI encoding)
	useAux bool
}

// NewModel creates a new render Model with the given scene, encoder, and options.
func NewModel(scene *Scene, encoder Encoder, caps TerminalCaps, fps int) *Model {
	m := &Model{
		Scene:   scene,
		encoder: encoder,
		caps:    caps,
		tier:    SelectTier(caps),
		width:   caps.Width,
		height:  caps.Height,
		fps:     fps,
	}

	// Check if encoder supports aux buffers
	if _, ok := encoder.(AuxEncoder); ok {
		m.useAux = true
	}

	// Initialize encoder
	_ = encoder.Init(caps)

	// Set up renderer at the encoder's preferred resolution
	pixW, pixH := encoder.InternalResolution(m.width, m.height)
	if pixW > 0 && pixH > 0 {
		m.renderer = NewRenderer(pixW, pixH)
	}

	return m
}

// tickCmd returns a Bubble Tea command that schedules the next frame.
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(m.fps), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init implements tea.Model. Starts the render loop.
func (m *Model) Init() tea.Cmd {
	m.lastTick = time.Now()
	return m.tickCmd()
}

// Update implements tea.Model. Handles tick, key, and resize messages.
// Architecture doc Section 6.2
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		now := time.Time(msg)
		dt := now.Sub(m.lastTick).Seconds()
		m.lastTick = now

		// Call user update callback
		if m.OnUpdate != nil {
			m.OnUpdate(dt)
		}

		// Advance animation state
		m.Scene.Tick(now)

		// Render frame
		if m.renderer != nil {
			if m.useAux {
				m.lastFrame, m.lastAux = m.renderer.RenderFrameWithAux(m.Scene)
			} else {
				m.lastFrame = m.renderer.RenderFrame(m.Scene)
			}

			// Encode to terminal output
			if m.lastFrame != nil {
				if m.useAux && m.lastAux != nil {
					if auxEnc, ok := m.encoder.(AuxEncoder); ok {
						m.output = auxEnc.EncodeWithAux(m.lastFrame, m.lastAux)
					} else {
						m.output = m.encoder.Encode(m.lastFrame)
					}
				} else {
					m.output = m.encoder.Encode(m.lastFrame)
				}
			}
		}

		return m, m.tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		// Forward to user key handler
		if m.OnKey != nil {
			m.OnKey(msg)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update caps
		m.caps.Width = m.width
		m.caps.Height = m.height

		// Re-init encoder with new dimensions
		_ = m.encoder.Init(m.caps)

		// Resize renderer to match new terminal dimensions
		pixW, pixH := m.encoder.InternalResolution(m.width, m.height)
		if pixW > 0 && pixH > 0 {
			if m.renderer != nil {
				m.renderer.Resize(pixW, pixH)
			} else {
				m.renderer = NewRenderer(pixW, pixH)
			}
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model. Returns the encoded frame as a string.
func (m *Model) View() string {
	return m.output
}

// SetEncoder changes the active encoder. Re-initializes with current caps.
func (m *Model) SetEncoder(enc Encoder) {
	m.encoder = enc
	_ = enc.Init(m.caps)

	// Check aux support
	_, m.useAux = enc.(AuxEncoder)

	// Resize renderer for new encoder
	pixW, pixH := enc.InternalResolution(m.width, m.height)
	if pixW > 0 && pixH > 0 {
		if m.renderer != nil {
			m.renderer.Resize(pixW, pixH)
		} else {
			m.renderer = NewRenderer(pixW, pixH)
		}
	}
}

// Tier returns the active output tier.
func (m *Model) Tier() OutputTier {
	return m.tier
}

// Caps returns the terminal capabilities.
func (m *Model) Caps() TerminalCaps {
	return m.caps
}
