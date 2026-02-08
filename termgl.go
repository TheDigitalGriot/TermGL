// Package termgl provides a two-tier 3D rendering system for terminal applications.
//
// It uses FauxGL as a software rasterizer and outputs to terminals via either
// Sixel pixel graphics (Tier 1) or ANSI subpixel character encoding (Tier 2).
// Both tiers share the same scene graph, camera, lighting, and animation system.
//
// Quick start with auto-detection:
//
//	mesh, _ := termgl.LoadOBJ("model.obj")
//	viewport := termgl.NewViewport(
//	    termgl.WithMesh(mesh),
//	    termgl.WithFPS(30),
//	    termgl.WithAutoDetect(),
//	)
//	p := tea.NewProgram(viewport, tea.WithAltScreen())
//	p.Run()
//
// Architecture doc Section 11
package termgl

import (
	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
	"github.com/charmbracelet/termgl/tier2"
	"github.com/fogleman/fauxgl"
)

// viewportConfig holds the configuration for NewViewport.
type viewportConfig struct {
	mesh      *fauxgl.Mesh
	meshPath  string
	fps       int
	encoder   render.Encoder
	caps      render.TerminalCaps
	autoDetect bool
	scene     *render.Scene
	onUpdate  func(dt float64)
}

// ViewportOption configures a viewport.
type ViewportOption func(*viewportConfig)

// NewViewport creates a new 3D viewport Bubble Tea model with functional options.
// Architecture doc Section 11
func NewViewport(opts ...ViewportOption) *render.Model {
	cfg := &viewportConfig{
		fps: 30,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Auto-detect terminal capabilities if requested
	if cfg.autoDetect || cfg.caps == (render.TerminalCaps{}) {
		cfg.caps = render.Detect()
	}

	// Auto-select encoder if none specified
	if cfg.encoder == nil {
		tier := render.SelectTier(cfg.caps)
		switch tier {
		case render.TierSixel:
			cfg.encoder = NewSixelEncoder()
		default:
			cfg.encoder = NewANSIEncoder()
		}
	}

	// Build scene
	scene := cfg.scene
	if scene == nil {
		scene = render.NewScene()

		// Add mesh if provided
		if cfg.mesh != nil {
			node := render.NewMeshNode(cfg.mesh)
			scene.Root.AddChild(node)
		}

		// Default lighting
		if len(scene.Lights) == 0 {
			light := render.NewDirectionalLight(0.5, 0.5, 1, fauxgl.Color{R: 1, G: 1, B: 1, A: 1}, 0.8)
			scene.Lights = append(scene.Lights, light)
			scene.Ambient = fauxgl.Color{R: 0.12, G: 0.12, B: 0.18, A: 1}
		}

		// Default camera
		scene.Camera.SetPosition(0, 0, 3)
		scene.Camera.SetTarget(0, 0, 0)
	}

	model := render.NewModel(scene, cfg.encoder, cfg.caps, cfg.fps)

	if cfg.onUpdate != nil {
		model.OnUpdate = cfg.onUpdate
	}

	return model
}

// --- Viewport options ---

// WithMesh sets the mesh to render in the viewport.
func WithMesh(mesh *fauxgl.Mesh) ViewportOption {
	return func(c *viewportConfig) {
		c.mesh = mesh
	}
}

// WithFPS sets the target frame rate.
func WithFPS(fps int) ViewportOption {
	return func(c *viewportConfig) {
		c.fps = fps
	}
}

// WithAutoDetect enables automatic terminal capability detection and tier selection.
func WithAutoDetect() ViewportOption {
	return func(c *viewportConfig) {
		c.autoDetect = true
	}
}

// WithEncoder sets a specific encoder (overrides auto-detection).
func WithEncoder(enc render.Encoder) ViewportOption {
	return func(c *viewportConfig) {
		c.encoder = enc
	}
}

// WithCaps sets the terminal capabilities explicitly (overrides auto-detection).
func WithCaps(caps render.TerminalCaps) ViewportOption {
	return func(c *viewportConfig) {
		c.caps = caps
	}
}

// WithScene sets a pre-built scene (overrides WithMesh).
func WithScene(scene *render.Scene) ViewportOption {
	return func(c *viewportConfig) {
		c.scene = scene
	}
}

// WithOnUpdate sets a callback called each frame before rendering.
func WithOnUpdate(fn func(dt float64)) ViewportOption {
	return func(c *viewportConfig) {
		c.onUpdate = fn
	}
}

// --- Sixel encoder options ---

// SixelOption configures a Sixel encoder.
type SixelOption func(*sixelConfig)

type sixelConfig struct {
	maxColors int
	dither    tier1.DitherMode
	colorSpace tier1.ColorSpace
	quantizer tier1.Quantizer
	rle       bool
	stable    bool
	adaptRate float64
	maxDrift  int
}

// NewSixelEncoder creates a Tier 1 Sixel encoder with functional options.
func NewSixelEncoder(opts ...SixelOption) *tier1.SixelOutput {
	cfg := &sixelConfig{
		maxColors:  256,
		dither:     tier1.DitherOrdered4x4,
		colorSpace: tier1.ColorSpaceRGB,
		rle:        true,
		stable:     true,
		adaptRate:  0.3,
		maxDrift:   32,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Build quantizer
	var quantizer tier1.Quantizer
	if cfg.quantizer != nil {
		quantizer = cfg.quantizer
	} else {
		octree := tier1.NewOctreeQuantizer(cfg.dither, cfg.colorSpace)
		if cfg.stable {
			quantizer = tier1.NewStablePaletteQuantizer(cfg.adaptRate, cfg.maxDrift, octree)
		} else {
			quantizer = octree
		}
	}

	return tier1.NewSixelOutput(quantizer, cfg.maxColors, cfg.rle)
}

// SixelColors sets the maximum palette size for Sixel encoding.
func SixelColors(n int) SixelOption {
	return func(c *sixelConfig) {
		c.maxColors = n
	}
}

// SixelDither sets the dithering mode for Sixel encoding.
func SixelDither(mode tier1.DitherMode) SixelOption {
	return func(c *sixelConfig) {
		c.dither = mode
	}
}

// SixelColorSpace sets the color space for quantization distance calculations.
func SixelColorSpace(cs tier1.ColorSpace) SixelOption {
	return func(c *sixelConfig) {
		c.colorSpace = cs
	}
}

// SixelQuantizer sets a custom quantizer (overrides dither/colorspace options).
func SixelQuantizer(q tier1.Quantizer) SixelOption {
	return func(c *sixelConfig) {
		c.quantizer = q
	}
}

// SixelStable controls whether palette stabilization is used for animation.
func SixelStable(enabled bool) SixelOption {
	return func(c *sixelConfig) {
		c.stable = enabled
	}
}

// SixelRLE controls run-length encoding in Sixel output.
func SixelRLE(enabled bool) SixelOption {
	return func(c *sixelConfig) {
		c.rle = enabled
	}
}

// --- ANSI encoder options ---

// ANSIOption configures an ANSI subpixel encoder.
type ANSIOption func(*ansiConfig)

type ansiConfig struct {
	blitter       tier2.Blitter
	frequencySplit bool
	edgeAware     bool
	delta         bool
	dither        tier1.DitherMode
}

// NewANSIEncoder creates a Tier 2 ANSI subpixel encoder with functional options.
func NewANSIEncoder(opts ...ANSIOption) *tier2.ANSIOutput {
	cfg := &ansiConfig{
		blitter:        tier2.SextantBlitter{},
		frequencySplit: true,
		edgeAware:      true,
		delta:          true,
		dither:         tier1.DitherOrdered4x4,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return tier2.NewANSIOutput(cfg.blitter).
		WithDither(cfg.dither).
		WithFrequencySplit(cfg.frequencySplit).
		WithEdgeAware(cfg.edgeAware).
		WithDeltaEncoding(cfg.delta)
}

// WithBlitter sets the blitter (character resolution) for ANSI encoding.
func WithBlitter(b tier2.Blitter) ANSIOption {
	return func(c *ansiConfig) {
		c.blitter = b
	}
}

// WithFrequencySplit controls Y/Cb/Cr frequency splitting.
func WithFrequencySplit(enabled bool) ANSIOption {
	return func(c *ansiConfig) {
		c.frequencySplit = enabled
	}
}

// WithDither sets the dithering mode for ANSI encoding.
func WithDither(mode tier1.DitherMode) ANSIOption {
	return func(c *ansiConfig) {
		c.dither = mode
	}
}

// WithDelta controls delta encoding (only emit changed cells).
func WithDelta(enabled bool) ANSIOption {
	return func(c *ansiConfig) {
		c.delta = enabled
	}
}

// WithEdgeAware controls edge-aware character selection using depth buffers.
func WithEdgeAware(enabled bool) ANSIOption {
	return func(c *ansiConfig) {
		c.edgeAware = enabled
	}
}

// --- Mesh loading convenience ---

// LoadOBJ loads an OBJ file and returns a FauxGL mesh.
func LoadOBJ(path string) (*fauxgl.Mesh, error) {
	return render.LoadOBJ(path)
}

// LoadSTL loads an STL file and returns a FauxGL mesh.
func LoadSTL(path string) (*fauxgl.Mesh, error) {
	return render.LoadSTL(path)
}
