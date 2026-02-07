// Raycast prism with spectral dispersion rendered with half-block pixels (▀).
//
// A white beam refracts through a slowly oscillating equilateral prism and
// disperses into spectral colors via wavelength-dependent Snell's law
// (Cauchy dispersion model). Every pixel is raycast each frame — no meshes.
//
// Run with: go run ./examples/prism
package main

import (
	"fmt"
	"image/color"
	stdmath "math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/framebuffer"
)

const (
	fps             = 30
	numSpectralRays = 32
)

// ============================================================================
// 2D Vector
// ============================================================================

type vec2 struct{ x, y float64 }

func v2(x, y float64) vec2          { return vec2{x, y} }
func add2(a, b vec2) vec2           { return vec2{a.x + b.x, a.y + b.y} }
func sub2(a, b vec2) vec2           { return vec2{a.x - b.x, a.y - b.y} }
func scale2(a vec2, s float64) vec2 { return vec2{a.x * s, a.y * s} }
func dot2(a, b vec2) float64        { return a.x*b.x + a.y*b.y }
func cross2(a, b vec2) float64      { return a.x*b.y - a.y*b.x }
func length2(a vec2) float64        { return stdmath.Sqrt(a.x*a.x + a.y*a.y) }

func normalize2(a vec2) vec2 {
	l := length2(a)
	if l < 1e-10 {
		return vec2{}
	}
	return vec2{a.x / l, a.y / l}
}

// ============================================================================
// Optics
// ============================================================================

// wavelengthToRGB converts a visible wavelength (380-700 nm) to linear RGB.
func wavelengthToRGB(wl float64) (r, g, b float64) {
	switch {
	case wl < 380 || wl > 700:
		return 0, 0, 0
	case wl < 440:
		r, b = -(wl-440)/(440-380), 1.0
	case wl < 490:
		g, b = (wl-440)/(490-440), 1.0
	case wl < 510:
		g, b = 1.0, -(wl-510)/(510-490)
	case wl < 580:
		r, g = (wl-510)/(580-510), 1.0
	case wl < 645:
		r, g = 1.0, -(wl-645)/(645-580)
	default:
		r = 1.0
	}
	// Intensity rolloff at spectrum edges.
	var f float64
	switch {
	case wl < 420:
		f = 0.3 + 0.7*(wl-380)/40
	case wl > 645:
		f = 0.3 + 0.7*(700-wl)/55
	default:
		f = 1.0
	}
	return r * f, g * f, b * f
}

// cauchyIndex returns the refractive index for a given wavelength (nm)
// using Cauchy's equation. Dispersion is exaggerated for visual clarity.
func cauchyIndex(wl float64) float64 {
	um := wl / 1000.0 // nm to um
	return 1.50 + 0.012/(um*um)
}

// refract2D computes the refracted direction via Snell's law.
// I is the unit incident direction, N is the outward surface normal.
func refract2D(I, N vec2, n1, n2 float64) (vec2, bool) {
	cosI := -dot2(I, N)
	if cosI < 0 {
		N = vec2{-N.x, -N.y}
		cosI = -cosI
	}
	eta := n1 / n2
	k := 1.0 - eta*eta*(1.0-cosI*cosI)
	if k < 0 {
		return vec2{}, false // total internal reflection
	}
	cosT := stdmath.Sqrt(k)
	return normalize2(vec2{
		x: eta*I.x + (eta*cosI-cosT)*N.x,
		y: eta*I.y + (eta*cosI-cosT)*N.y,
	}), true
}

// ============================================================================
// Geometry helpers
// ============================================================================

// prismVertices returns the 3 vertices of an equilateral triangle centred at
// (cx,cy) with the given side length and rotation angle (radians, 0 = apex up).
func prismVertices(cx, cy, side, angle float64) [3]vec2 {
	r := side / stdmath.Sqrt(3) // circumradius
	var v [3]vec2
	for i := 0; i < 3; i++ {
		a := angle + float64(i)*2*stdmath.Pi/3 - stdmath.Pi/2
		v[i] = vec2{cx + r*stdmath.Cos(a), cy + r*stdmath.Sin(a)}
	}
	return v
}

func pointInTriangle(p, a, b, c vec2) bool {
	d1 := cross2(sub2(b, a), sub2(p, a))
	d2 := cross2(sub2(c, b), sub2(p, b))
	d3 := cross2(sub2(a, c), sub2(p, c))
	return !((d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0))
}

// outwardNormal returns the unit outward normal of edge A->B, given a point
// known to be interior to the triangle.
func outwardNormal(a, b, interior vec2) vec2 {
	edge := sub2(b, a)
	n := vec2{edge.y, -edge.x}
	mid := scale2(add2(a, b), 0.5)
	if dot2(n, sub2(interior, mid)) > 0 {
		n = vec2{-edge.y, edge.x}
	}
	return normalize2(n)
}

// raySegIntersect returns the t parameter along ray O+t*D where it crosses
// segment A->B.  Returns (t, true) when valid (t>0, s in [0,1]).
func raySegIntersect(O, D, A, B vec2) (float64, bool) {
	E := sub2(B, A)
	F := sub2(A, O)
	denom := cross2(D, E)
	if stdmath.Abs(denom) < 1e-10 {
		return 0, false
	}
	t := cross2(F, E) / denom
	s := -cross2(D, F) / denom
	return t, t > 1e-6 && s >= 0 && s <= 1
}

func distToSegment(p, a, b vec2) float64 {
	ab := sub2(b, a)
	ap := sub2(p, a)
	l2 := dot2(ab, ab)
	if l2 < 1e-10 {
		return length2(ap)
	}
	t := clamp01(dot2(ap, ab) / l2)
	proj := add2(a, scale2(ab, t))
	return length2(sub2(p, proj))
}

func distToTriangleEdge(p vec2, v [3]vec2) float64 {
	d0 := distToSegment(p, v[0], v[1])
	d1 := distToSegment(p, v[1], v[2])
	d2 := distToSegment(p, v[2], v[0])
	return stdmath.Min(d0, stdmath.Min(d1, d2))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// ============================================================================
// Spectral ray tracing
// ============================================================================

type tracedRay struct {
	r, g, b     float64
	entry, exit vec2
	internalDir vec2
	exitDir     vec2
	valid       bool
}

// traceSpectralRays traces numSpectralRays wavelengths through the prism and
// returns the entry/exit points and exit direction for each.
func traceSpectralRays(beamOrigin, beamDir vec2, verts [3]vec2) []tracedRay {
	center := vec2{
		(verts[0].x + verts[1].x + verts[2].x) / 3,
		(verts[0].y + verts[1].y + verts[2].y) / 3,
	}
	edges := [3][2]int{{0, 1}, {1, 2}, {2, 0}}

	rays := make([]tracedRay, numSpectralRays)
	for i := range rays {
		wl := 380.0 + float64(i)*(700.0-380.0)/float64(numSpectralRays-1)
		cr, cg, cb := wavelengthToRGB(wl)
		n := cauchyIndex(wl)

		// Find beam -> prism entry (closest edge intersection).
		bestT := stdmath.MaxFloat64
		entryEdge := -1
		var entry vec2
		for ei, e := range edges {
			t, ok := raySegIntersect(beamOrigin, beamDir, verts[e[0]], verts[e[1]])
			if ok && t < bestT {
				bestT = t
				entryEdge = ei
				entry = add2(beamOrigin, scale2(beamDir, t))
			}
		}
		if entryEdge < 0 {
			continue
		}

		// Refract into prism.
		e := edges[entryEdge]
		entryN := outwardNormal(verts[e[0]], verts[e[1]], center)
		intDir, ok := refract2D(beamDir, entryN, 1.0, n)
		if !ok {
			continue
		}

		// Trace internal ray to exit edge.
		bestT = stdmath.MaxFloat64
		exitEdge := -1
		var exit vec2
		for ei, e2 := range edges {
			if ei == entryEdge {
				continue
			}
			t, ok := raySegIntersect(entry, intDir, verts[e2[0]], verts[e2[1]])
			if ok && t < bestT {
				bestT = t
				exitEdge = ei
				exit = add2(entry, scale2(intDir, t))
			}
		}
		if exitEdge < 0 {
			continue
		}

		// Refract out of prism.
		e2 := edges[exitEdge]
		exitN := outwardNormal(verts[e2[0]], verts[e2[1]], center)
		exitDir, ok := refract2D(intDir, exitN, n, 1.0)
		if !ok {
			continue // total internal reflection
		}

		rays[i] = tracedRay{
			r: cr, g: cg, b: cb,
			entry: entry, exit: exit,
			internalDir: intDir, exitDir: exitDir,
			valid: true,
		}
	}
	return rays
}

// ============================================================================
// Bubble Tea model
// ============================================================================

type tickMsg time.Time

type model struct {
	fb      *framebuffer.Framebuffer
	frame   int
	vw, vh  int
	gridW   int
	gridH   int
	renderH int
}

func initialModel() model {
	caps := detect.Capabilities()
	gridW := caps.GridSize.X
	gridH := caps.GridSize.Y
	renderH := gridH - 1
	if renderH < 1 {
		renderH = 1
	}
	vw := gridW
	vh := renderH * 2

	return model{
		fb:      framebuffer.New(framebuffer.WithFixedSize(vw, vh)),
		vw:      vw,
		vh:      vh,
		gridW:   gridW,
		gridH:   gridH,
		renderH: renderH,
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

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

	case tickMsg:
		m.frame++
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	renderFrame(&m)
	out := framebufferToString(m.fb, m.gridW, m.renderH)
	return out + "\n\x1b[0m[q] quit"
}

// ============================================================================
// Per-frame rendering
// ============================================================================

func renderFrame(m *model) {
	if m.vw <= 0 || m.vh <= 0 {
		return
	}

	t := float64(m.frame) / float64(fps)
	aspect := float64(m.vw) / float64(m.vh)

	// Prism: equilateral triangle, gently oscillating.
	prismCX := aspect * 0.38
	prismCY := 0.5
	prismSide := 0.35
	prismAngle := 0.12*stdmath.Sin(t*0.4) + 0.05*stdmath.Sin(t*0.67)
	verts := prismVertices(prismCX, prismCY, prismSide, prismAngle)

	// Beam: horizontal from far left at prism centre height.
	beamOrigin := v2(-1.0, prismCY)
	beamDir := v2(1.0, 0.0)

	// Trace spectral rays through the prism.
	rays := traceSpectralRays(beamOrigin, beamDir, verts)
	midRay := rays[numSpectralRays/2]

	// Shade every pixel.
	invVH := 1.0 / float64(m.vh)
	for py := 0; py < m.vh; py++ {
		sy := float64(py) * invVH
		for px := 0; px < m.vw; px++ {
			sx := float64(px) * invVH // same scale as sy for square pixels
			col := shadePixel(sx, sy, aspect, verts, rays, midRay, beamOrigin, m.frame)
			m.fb.SetPixel(px, py, col)
		}
	}
}

func shadePixel(sx, sy, aspect float64, verts [3]vec2, rays []tracedRay, midRay tracedRay, beamOrigin vec2, frame int) color.RGBA {
	var r, g, b float64
	p := v2(sx, sy)

	// --- background: dark with vignette ---
	mx := aspect * 0.5
	dx := (sx - mx) / aspect
	dy := sy - 0.5
	vig := 1.0 - clampf((dx*dx+dy*dy)*2.5, 0, 0.6)
	r = 0.008 * vig
	g = 0.008 * vig
	b = 0.022 * vig

	inPrism := pointInTriangle(p, verts[0], verts[1], verts[2])

	if !inPrism {
		// --- incoming white beam ---
		if midRay.valid {
			beamDist := stdmath.Abs(sy - beamOrigin.y)
			bw := 0.012
			core := stdmath.Exp(-beamDist * beamDist / (2 * bw * bw * 0.09))
			halo := stdmath.Exp(-beamDist * beamDist / (2 * bw * bw))
			fade := clampf((midRay.entry.x-sx)*20, 0, 1)
			beam := (core*0.6 + halo*0.4) * fade
			r += beam * 0.95
			g += beam * 0.95
			b += beam * 1.0
		}

		// --- dispersed spectral rays ---
		for _, ray := range rays {
			if !ray.valid {
				continue
			}
			v := sub2(p, ray.exit)
			along := dot2(v, ray.exitDir)
			if along < -0.005 {
				continue
			}
			perp := sub2(v, scale2(ray.exitDir, along))
			perpDist := length2(perp)
			rw := 0.005 + along*0.002 // beam widens with distance
			glow := stdmath.Exp(-perpDist * perpDist / (2 * rw * rw))
			fadeIn := clampf(along*8, 0, 1)
			glow *= fadeIn

			r += glow * ray.r * 0.55
			g += glow * ray.g * 0.55
			b += glow * ray.b * 0.55
		}
	} else {
		// --- prism glass body ---
		r = r*0.25 + 0.012
		g = g*0.25 + 0.014
		b = b*0.25 + 0.035

		// Internal beam (white-ish, along mid-wavelength path).
		if midRay.valid {
			intLen := length2(sub2(midRay.exit, midRay.entry))
			if intLen > 0.001 {
				intDir := normalize2(sub2(midRay.exit, midRay.entry))
				v := sub2(p, midRay.entry)
				along := dot2(v, intDir)
				if along >= -0.01 && along <= intLen+0.01 {
					perp := sub2(v, scale2(intDir, along))
					pd := length2(perp)
					bw := 0.010
					glow := stdmath.Exp(-pd * pd / (2 * bw * bw))
					r += glow * 0.25
					g += glow * 0.25
					b += glow * 0.30
				}
			}
		}

		// Caustic shimmer inside glass.
		s1 := stdmath.Sin(sx*80+sy*60+float64(frame)*0.15)*0.5 + 0.5
		s2 := stdmath.Sin(sx*30-sy*45+float64(frame)*0.08)*0.5 + 0.5
		shimmer := s1 * s2 * 0.015
		r += shimmer * 0.5
		g += shimmer * 0.6
		b += shimmer * 1.0
	}

	// --- Fresnel-like prism edge glow (visible from both sides) ---
	edgeDist := distToTriangleEdge(p, verts)
	ew := 0.005
	edgeGlow := stdmath.Exp(-edgeDist * edgeDist / (2 * ew * ew))
	r += edgeGlow * 0.18
	g += edgeGlow * 0.20
	b += edgeGlow * 0.30

	// Clamp.
	r = clampf(r, 0, 1)
	g = clampf(g, 0, 1)
	b = clampf(b, 0, 1)

	// Approximate sRGB gamma for brighter output on terminals.
	r = stdmath.Pow(r, 0.85)
	g = stdmath.Pow(g, 0.85)
	b = stdmath.Pow(b, 0.85)

	return color.RGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}

// ============================================================================
// Half-block encoder (framebuffer -> ANSI string)
// ============================================================================

func framebufferToString(fb *framebuffer.Framebuffer, gridW, renderH int) string {
	var buf strings.Builder
	buf.Grow(gridW * renderH * 30)

	var lastFG, lastBG color.RGBA
	for row := 0; row < renderH; row++ {
		first := true
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

			if first || top != lastFG {
				fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm", top.R, top.G, top.B)
				lastFG = top
			}
			if first || bot != lastBG {
				fmt.Fprintf(&buf, "\x1b[48;2;%d;%d;%dm", bot.R, bot.G, bot.B)
				lastBG = bot
			}
			buf.WriteString("▀")
			first = false
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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
