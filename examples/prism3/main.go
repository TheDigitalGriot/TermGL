// Compact raycast prism with artifact-free rendering, suitable for TUI headers.
//
// Based on the JSX reference (generatePrismScene) with fixes for sub-pixel
// noise artifacts caused by Gaussian glow tails. Renders in a configurable
// number of rows (default 3) so it can serve as a decorative header widget.
//
// Run with: go run ./examples/prism3
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
	fps        = 30
	headerRows = 3 // cell rows used for rendering (change to 0 for full terminal)
)

// ============================================================================
// 2D helpers
// ============================================================================

type vec2 struct{ x, y float64 }

func pointInTriangle(px, py float64, v [3]vec2) bool {
	sign := func(p, a, b vec2) float64 {
		return (p.x-b.x)*(a.y-b.y) - (a.x-b.x)*(p.y-b.y)
	}
	p := vec2{px, py}
	d1 := sign(p, v[0], v[1])
	d2 := sign(p, v[1], v[2])
	d3 := sign(p, v[2], v[0])
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

func distToSeg(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	len2 := dx*dx + dy*dy
	var t float64
	if len2 > 0 {
		t = ((px-ax)*dx + (py-ay)*dy) / len2
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := ax+t*dx, ay+t*dy
	ex, ey := px-cx, py-cy
	return stdmath.Sqrt(ex*ex + ey*ey)
}

func distToTriEdge(px, py float64, v [3]vec2) float64 {
	best := stdmath.MaxFloat64
	for i := 0; i < 3; i++ {
		j := (i + 1) % 3
		d := distToSeg(px, py, v[i].x, v[i].y, v[j].x, v[j].y)
		if d < best {
			best = d
		}
	}
	return best
}

func lerpf(a, b, t float64) float64 { return a + (b-a)*t }

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ============================================================================
// Band color palette (matches JSX reference)
// ============================================================================

var bandColors = [4][3]float64{
	{59, 130, 246},  // blue
	{20, 184, 166},  // teal
	{34, 197, 94},   // green
	{245, 158, 11},  // amber
}

func bandLerp(t float64) (float64, float64, float64) {
	t = clampf(t, 0, 1)
	ct := t * 3.0
	i := int(ct)
	f := ct - float64(i)
	if i >= 3 {
		i = 3
		f = 0
	}
	j := i + 1
	if j > 3 {
		j = 3
	}
	a, b := bandColors[i], bandColors[j]
	return lerpf(a[0], b[0], f), lerpf(a[1], b[1], f), lerpf(a[2], b[2], f)
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
	if headerRows > 0 && headerRows < renderH {
		renderH = headerRows
	}
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
		if headerRows > 0 && headerRows < m.renderH {
			m.renderH = headerRows
		}
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
// Per-frame rendering — JSX port with artifact fixes
// ============================================================================

func renderFrame(m *model) {
	w := float64(m.vw)
	h := float64(m.vh)
	if w <= 0 || h <= 0 {
		return
	}

	t := float64(m.frame) / float64(fps)
	cx := w / 2
	cy := h / 2

	minDim := w
	if h < minDim {
		minDim = h
	}
	s := minDim / 30.0
	if s < 0.1 {
		s = 0.1
	}

	// ── Prism geometry ──
	prismAngle := t * 0.2
	prismSize := minDim * 0.35 // larger ratio so the prism reads at compact sizes
	var tri [3]vec2
	for i := 0; i < 3; i++ {
		a := prismAngle + float64(i)/3.0*2*stdmath.Pi - stdmath.Pi/2
		tri[i] = vec2{cx + stdmath.Cos(a)*prismSize, cy + stdmath.Sin(a)*prismSize}
	}

	// ── Beam parameters ──
	beamY := cy + stdmath.Sin(t*0.3)*h*0.05
	beamEntryX := cx - prismSize*0.3
	beamExitX := cx + prismSize*0.4
	disperseAngle := 0.55 + stdmath.Sin(t*0.15)*0.08

	// Minimum glow contribution threshold — any single-channel addition
	// below this is invisible noise and gets suppressed.
	const glowFloor = 2.0

	for py := 0; py < m.vh; py++ {
		y := float64(py)
		for px := 0; px < m.vw; px++ {
			x := float64(px)

			// --- background with radial vignette ---
			vx := (x/w - 0.5) * 2
			vy := (y/h - 0.5) * 2
			vig := 1.0 - 0.4*(vx*vx+vy*vy)
			r := 8.0 * vig
			g := 6.0 * vig
			b := 16.0 * vig

			// === Incoming white beam (left of prism) ===
			if x < beamEntryX+4*s {
				beamDist := stdmath.Abs(y - beamY)
				beamWidth := (1.5 + x/w*1.5) * s
				if beamDist < beamWidth {
					beamI := stdmath.Max(0, 1-beamDist/beamWidth)
					bi := beamI * beamI * (0.7 + 0.3*(x/beamEntryX))
					r += 220 * bi
					g += 215 * bi
					b += 240 * bi
				}
				// soft glow — tightened range (was *5, now *3)
				if beamDist < beamWidth*3 {
					glow := stdmath.Exp(-beamDist*beamDist/(beamWidth*beamWidth*6)) * 0.15
					gr, gg, gb := 120*glow, 115*glow, 150*glow
					if gr >= glowFloor || gg >= glowFloor || gb >= glowFloor {
						r += gr
						g += gg
						b += gb
					}
				}
			}

			// === Dispersed rainbow (right of prism) ===
			if x > beamExitX-4*s {
				dx := x - beamExitX
				rightSpan := w - beamExitX
				progress := 0.0
				if rightSpan > 0 {
					progress = stdmath.Min(1, dx/rightSpan)
				}
				spreadY := disperseAngle * dx
				bandHeight := spreadY * 2 / 4

				if bandHeight > 0.3*s {
					for band := 0; band < 4; band++ {
						bandCenterY := beamY - spreadY + bandHeight*(float64(band)+0.5)
						bandDist := stdmath.Abs(y - bandCenterY)
						bw := bandHeight*0.55 + progress*0.8*s
						cr, cg, cb := bandLerp(float64(band) / 3.0)

						if bandDist < bw {
							intensity := stdmath.Max(0, 1-bandDist/bw)
							ii := intensity * intensity * (0.5 + 0.5*stdmath.Min(1, dx/(8*s)))
							r += cr * ii
							g += cg * ii
							b += cb * ii
						}
						// band glow — tightened range (was *3.5, now *2.5)
						// with noise floor check
						if bandDist < bw*2.5 {
							glow := stdmath.Exp(-bandDist*bandDist/(bw*bw*4)) * 0.08 * (0.3 + progress*0.7)
							gr, gg, gb := cr*glow, cg*glow, cb*glow
							if gr >= glowFloor || gg >= glowFloor || gb >= glowFloor {
								r += gr
								g += gg
								b += gb
							}
						}
					}
				}
			}

			// === Glass prism ===
			inside := pointInTriangle(x, y, tri)
			edgeDist := distToTriEdge(x, y, tri)

			if inside {
				depth := edgeDist / prismSize
				glassR := 30.0 + 40.0*depth
				glassG := 35.0 + 50.0*depth
				glassB := 55.0 + 80.0*depth

				caustic := stdmath.Sin(x*0.3/s+y*0.2/s+t*1.5)*
					stdmath.Sin(x*0.15/s-y*0.25/s+t*0.8)*0.5 + 0.5
				ci := caustic * depth * 0.4

				alpha := 0.55 + 0.2*(1-depth)
				r = r*(1-alpha) + (glassR+30*ci)*alpha
				g = g*(1-alpha) + (glassG+20*ci)*alpha
				b = b*(1-alpha) + (glassB+50*ci)*alpha

				angle := stdmath.Atan2(y-cy, x-cx)
				spectrum := (angle/stdmath.Pi + 1) * 0.5
				sr, sg, sb := bandLerp(spectrum)
				specI := 0.12 * depth
				r += sr * specI
				g += sg * specI
				b += sb * specI
			}

			// edge highlight (Fresnel-like rim)
			edgeThresh := 2.5 * s
			if edgeDist < edgeThresh {
				edgeI := stdmath.Max(0, 1-edgeDist/edgeThresh)
				ei := edgeI * edgeI * 0.8
				er, eg, eb := 140*ei, 150*ei, 200*ei
				if er >= glowFloor || eg >= glowFloor || eb >= glowFloor {
					r += er
					g += eg
					b += eb
				}
			}

			// specular highlight on top facet
			if inside {
				topDist := distToSeg(x, y, tri[0].x, tri[0].y, tri[1].x, tri[1].y)
				if topDist < prismSize*0.15 {
					si := stdmath.Exp(-topDist*topDist/(prismSize*prismSize*0.003)) * 0.6
					r += 200 * si
					g += 210 * si
					b += 255 * si
				}
			}

			r = clampf(r, 0, 255)
			g = clampf(g, 0, 255)
			b = clampf(b, 0, 255)
			m.fb.SetPixel(px, py, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
		}
	}
}

// ============================================================================
// Half-block encoder
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
