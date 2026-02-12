package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
	"github.com/fogleman/fauxgl"
	"golang.org/x/term"
)

func main() {
	// Load Suzanne mesh
	mesh, err := render.LoadOBJ("models/suzanne_blender_monkey.obj")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mesh: %v\n", err)
		os.Exit(1)
	}

	// Normalize mesh to fit in unit cube
	mesh.BiUnitCube()

	// Create scene
	scene := render.NewScene()

	// Add mesh to scene
	meshNode := render.NewMeshNode(mesh)
	scene.Root.AddChild(meshNode)

	// Set up camera — pull back a bit to leave room for text above
	scene.Camera.SetPosition(0, 0.4, 3)
	scene.Camera.SetTarget(0, 0.4, 0)

	// Add a directional light
	light := render.NewDirectionalLight(0.5, 0.5, 1, fauxgl.Color{R: 1, G: 1, B: 1, A: 1}, 0.8)
	scene.Lights = append(scene.Lights, light)

	// Set ambient light
	scene.Ambient = fauxgl.Color{R: 0.15, G: 0.15, B: 0.2, A: 1}

	// Detect terminal size for full-width rendering
	termWidth, termHeight, termErr := term.GetSize(int(os.Stdout.Fd()))
	if termErr != nil {
		termWidth, termHeight = 120, 30
	}

	// Compute pixel dimensions from terminal cell size, at half resolution
	// for performance (full res is too many pixels to quantize+encode at 30fps)
	cellW, cellH := 8, 16
	width := termWidth * cellW / 2
	height := (termHeight - 2) * cellH / 2

	renderer := render.NewRenderer(width, height)

	// Create Sixel pipeline
	// Use OctreeQuantizer with Bayer 4x4 dithering (best for animation)
	octreeQuant := tier1.NewOctreeQuantizer(tier1.DitherOrdered4x4, tier1.ColorSpaceRGB)

	// Wrap in StablePaletteQuantizer for flicker-free animation
	stableQuant := tier1.NewStablePaletteQuantizer(0.3, 32, octreeQuant)

	// Create SixelOutput
	sixelOut := tier1.NewSixelOutput(stableQuant, 256, true)

	// Initialize with actual terminal caps
	caps := render.TerminalCaps{
		Sixel:          true,
		SixelMaxColors: 256,
		CellWidth:      cellW,
		CellHeight:     cellH,
		Width:          termWidth,
		Height:         termHeight,
	}
	if err := sixelOut.Init(caps); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing SixelOutput: %v\n", err)
		os.Exit(1)
	}

	// Hide cursor, clear screen, enable DECSDM (no scroll after Sixel)
	fmt.Print("\x1b[?25l\x1b[?80h\x1b[2J\x1b[H")

	defer func() {
		// Disable DECSDM, show cursor, reset colors
		fmt.Print("\x1b[?80l\x1b[?25h\x1b[0m")
	}()

	// Column where the right panel starts (after the Sixel image)
	imgCols := width / cellW
	panelCol := imgCols + 2
	panelWidth := termWidth - panelCol - 1
	if panelWidth < 30 {
		panelWidth = 30
	}

	// Lipgloss styles
	accent := lipgloss.Color("63")
	dim := lipgloss.Color("240")
	bright := lipgloss.Color("252")

	dimStyle := lipgloss.NewStyle().Foreground(dim)
	sectionStyle := lipgloss.NewStyle().Foreground(accent).Bold(true).Underline(true)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(panelWidth)

	tableHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
	tableCellStyle := lipgloss.NewStyle().Foreground(bright)
	tableBorderStyle := lipgloss.NewStyle().Foreground(dim)

	// TODO: Canvas border (commented out — Sixel overwrites text cells and
	// cursor positioning is ignored by the terminal for Sixel data)
	// borderColor := "\x1b[38;5;63m"
	// borderReset := "\x1b[0m"
	// hBar := strings.Repeat("─", imgCols)
	// canvasTopBorder := fmt.Sprintf("\x1b[1;1H%s╭%s╮%s", borderColor, hBar, borderReset)
	// canvasBotBorder := fmt.Sprintf("\x1b[%d;1H%s╰%s╯%s", boxRows, borderColor, hBar, borderReset)
	// canvasSideBorders := ...

	// Animation loop
	const targetFPS = 30
	frameDuration := time.Second / targetFPS
	frameCount := 0
	startTime := time.Now()
	lastFPSReport := startTime
	var lastFPS float64
	var lastSixelSize int
	panelDirty := true

	for {
		frameStart := time.Now()

		// Rotate the mesh
		angle := float64(frameCount) * 0.03
		meshNode.SetRotation(0.2, angle, 0)
		scene.Tick(time.Now())

		// Render frame
		img := renderer.RenderFrame(scene)

		// Composite bubble background behind the 3D model
		t := time.Since(startTime).Seconds()
		compositeWithBubbleBackground(img, t, scene.Ambient)

		// Draw animated "TermGL" text above Suzanne
		drawAnimatedText(img, "TermGL", t)

		// Encode to Sixel at top-left (image fills the full box area)
		output := sixelOut.EncodeWithCursor(img, 1, 1)

		// Write to stdout
		fmt.Print(output)

		// TODO: Canvas border redraw (commented out — see note above)
		// fmt.Print(canvasTopBorder)
		// for _, s := range canvasSideBorders { fmt.Print(s) }
		// fmt.Print(canvasBotBorder)

		frameCount++
		lastSixelSize = len(output)

		// Update stats every 2 seconds
		now := time.Now()
		if now.Sub(lastFPSReport) >= 2*time.Second {
			elapsed := now.Sub(startTime).Seconds()
			lastFPS = float64(frameCount) / elapsed
			lastFPSReport = now
			panelDirty = true
		}

		// Render panel to the right of the Sixel image (only when data changes)
		if panelDirty && lastFPS > 0 {
			panelDirty = false

			statsTable := table.New().
				Headers("Metric", "Value").
				Row("FPS", fmt.Sprintf("%.1f", lastFPS)).
				Row("Frame", fmt.Sprintf("%d", frameCount)).
				Row("Resolution", fmt.Sprintf("%dx%d", width, height)).
				Row("Sixel Size", fmt.Sprintf("%d KB", lastSixelSize/1024)).
				Row("Terminal", fmt.Sprintf("%dx%d", termWidth, termHeight)).
				Border(lipgloss.RoundedBorder()).
				BorderStyle(tableBorderStyle).
				StyleFunc(func(row, col int) lipgloss.Style {
					if row == table.HeaderRow {
						return tableHeaderStyle
					}
					return tableCellStyle
				})

			pipelineTable := table.New().
				Headers("Setting", "Value").
				Row("Encoder", "Sixel (Tier 1)").
				Row("Quantizer", "Octree + Stable").
				Row("Dither", "Bayer 4x4").
				Row("Palette", "256 colors").
				Row("RLE", "Enabled").
				Border(lipgloss.RoundedBorder()).
				BorderStyle(tableBorderStyle).
				StyleFunc(func(row, col int) lipgloss.Style {
					if row == table.HeaderRow {
						return tableHeaderStyle
					}
					return tableCellStyle
				})

			panel := panelStyle.Render(strings.Join([]string{
				sectionStyle.Render("Performance"),
				"",
				statsTable.Render(),
				"",
				sectionStyle.Render("Pipeline"),
				"",
				pipelineTable.Render(),
				"",
				dimStyle.Render("Press Ctrl+C to exit"),
			}, "\n"))

			lines := strings.Split(panel, "\n")
			for i, line := range lines {
				if i+1 < termHeight {
					fmt.Printf("\x1b[%d;%dH%s", i+1, panelCol, line)
				}
			}
		}

		// Frame pacing
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}

		// Run for 300 frames (~10 seconds at 30 FPS)
		if frameCount >= 300 {
			break
		}
	}

	// Final stats
	elapsed := time.Since(startTime).Seconds()
	fps := float64(frameCount) / elapsed
	fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K Final: %d frames in %.1fs = %.1f FPS (%dx%d)\n",
		termHeight, frameCount, elapsed, fps, width, height)
}

// Logo from ref/branding/termgl-logo.md parsed as half-block art.
// Each character maps to a 1x2 sub-pixel pattern:
//   █ = top on, bottom on
//   ▀ = top on, bottom off
//   ▄ = top off, bottom on
//   space = both off
var logoLines = []string{
	"  ▄▄▄▄▄▄▄                  ▄   ▄▄▄▄   ▄▄▄    ",
	" █▀▀██▀▀▀▀                 ▀██████▀  ▀██▀    ",
	"    ██         ▄    ▄        ██   ▄   ██     ",
	"    ██   ▄█▀█▄ ████▄███▄███▄ ██  ██   ██     ",
	"    ██   ██▄█▀ ██   ██ ██ ██ ██  ██   ██     ",
	"    ▀██▄▄▀█▄▄▄▄█▀  ▄██ ██ ▀█ ▀█████  ████████",
	"                             ▄   ██          ",
	"                             ▀████▀          ",
}

var logoBitmap [][]bool
var logoBitmapW, logoBitmapH int

func init() {
	// Find max width in runes
	maxW := 0
	for _, line := range logoLines {
		runes := []rune(line)
		if len(runes) > maxW {
			maxW = len(runes)
		}
	}

	logoBitmapH = len(logoLines) * 2
	logoBitmapW = maxW
	logoBitmap = make([][]bool, logoBitmapH)
	for i := range logoBitmap {
		logoBitmap[i] = make([]bool, logoBitmapW)
	}

	for lineIdx, line := range logoLines {
		topRow := lineIdx * 2
		botRow := lineIdx * 2 + 1
		for col, ch := range []rune(line) {
			switch ch {
			case '█':
				logoBitmap[topRow][col] = true
				logoBitmap[botRow][col] = true
			case '▀':
				logoBitmap[topRow][col] = true
			case '▄':
				logoBitmap[botRow][col] = true
			}
		}
	}
}

// drawAnimatedText draws the TermGL logo onto the image with wave + blue shading.
func drawAnimatedText(img *image.NRGBA, _ string, t float64) {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	// Scale so the logo is about 60% of image width
	scale := imgW * 6 / (10 * logoBitmapW)
	if scale < 1 {
		scale = 1
	}

	totalW := logoBitmapW * scale
	totalH := logoBitmapH * scale

	// Center horizontally, place in top region
	startX := (imgW - totalW) / 2
	baseY := imgH/12 - totalH/4

	for row := 0; row < logoBitmapH; row++ {
		for col := 0; col < logoBitmapW; col++ {
			if !logoBitmap[row][col] {
				continue
			}

			// Per-column wave offset
			waveY := int(math.Sin(t*2.5+float64(col)*0.15) * float64(scale) * 1.2)

			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					x := startX + col*scale + dx
					y := baseY + waveY + row*scale + dy
					if x < 0 || x >= imgW || y < 0 || y >= imgH {
						continue
					}
					// Sample background and tint to accent color (ANSI 63 ≈ RGB 95,95,255)
					bg := img.NRGBAAt(x, y)
					lum := 0.299*float64(bg.R) + 0.587*float64(bg.G) + 0.114*float64(bg.B)
					brightness := clampf(0.4+lum/255.0*0.6, 0.4, 1.0)
					r := clampf(95*brightness, 0, 255)
					g := clampf(95*brightness, 0, 255)
					b := clampf(255*brightness, 0, 255)
					img.SetNRGBA(x, y, color.NRGBA{
						R: uint8(r), G: uint8(g), B: uint8(b), A: 255,
					})
				}
			}
		}
	}
}

// compositeWithBubbleBackground renders the GERP effect7-style bubble/orb
// background into pixels that match the ambient clear color (no geometry drawn).
// Uses Mandelbrot-mapped voronoi with 1/d² falloff for the dark-background
// pink/magenta look from the GERP 2025 title screen.
func compositeWithBubbleBackground(img *image.NRGBA, t float64, ambient fauxgl.Color) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Pre-compute the ambient clear color as uint8 for fast comparison
	ambR := uint8(clampf(ambient.R*255, 0, 255))
	ambG := uint8(clampf(ambient.G*255, 0, 255))
	ambB := uint8(clampf(ambient.B*255, 0, 255))

	for y := 0; y < h; y++ {
		py := (-1.0*float64(h) + 2.0*(float64(y)+0.5)) / float64(h)
		for x := 0; x < w; x++ {
			// Check if this pixel is the ambient clear color (no geometry)
			c := img.NRGBAAt(x, y)
			dr := int(c.R) - int(ambR)
			dg := int(c.G) - int(ambG)
			db := int(c.B) - int(ambB)
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if dr > 1 || dg > 1 || db > 1 {
				continue // geometry pixel, keep it
			}

			px := (-1.0*float64(w) + 2.0*(float64(x)+0.5)) / (2.0 * float64(h))

			// Value noise for warping
			vp := [2]float64{px, py}
			vp[0] += 8.0 * math.Sin(t*(math.Pi/3.0/8.0))
			vp[1] += 8.0 * math.Sin(t*(math.Pi/3.0*0.707/8.0))
			n := vnoise(vp)

			// Mandelbrot mapping (swap x,y like the GERP code)
			mp := [2]float64{py, px}
			mp[0] -= 0.25
			mp[0] *= 0.4
			mp[1] *= 0.4
			mmp := mandelmap(mp, mp)

			ml := math.Sqrt(mmp[0]*mmp[0] + mmp[1]*mmp[1])

			var cr, cg, cb float64
			if ml < 2.0 {
				// Voronoi cell: find nearest integer point
				pp := [2]float64{mmp[0], mmp[1]}
				pp[0] *= (0.5 + n)
				pp[1] *= (0.5 + n)
				pp[0] += 0.13 * t

				// Fractional part (distance to nearest cell center)
				npx := math.Round(pp[0])
				npy := math.Round(pp[1])
				cp := [2]float64{pp[0] - npx, pp[1] - npy}

				d := math.Sqrt(cp[0]*cp[0]+cp[1]*cp[1]) - 0.5*n
				// 1/d² falloff: bright at center, dark everywhere else
				intensity := 0.001 / math.Max(d*d, 0.0001)
				cr, cg, cb = bubblePalette(d + t)
				cr *= intensity
				cg *= intensity
				cb *= intensity
			}
			// Outside Mandelbrot set: stays black (cr,cg,cb = 0)

			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clampf(cr*255, 0, 255)),
				G: uint8(clampf(cg*255, 0, 255)),
				B: uint8(clampf(cb*255, 0, 255)),
				A: 255,
			})
		}
	}
}

// mandelmap iterates the Mandelbrot function 8 times: z = z² + c
func mandelmap(p, c [2]float64) [2]float64 {
	z := p
	for i := 0; i < 8; i++ {
		zx := z[0]*z[0] - z[1]*z[1] + c[0]
		zy := 2*z[0]*z[1] + c[1]
		z[0] = zx
		z[1] = zy
	}
	return z
}

// vnoise is a 2D value noise function (same as GERP's vnoise)
func vnoise(p [2]float64) float64 {
	ix := math.Floor(p[0])
	iy := math.Floor(p[1])
	fx := p[0] - ix
	fy := p[1] - iy

	// Smoothstep interpolation
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)

	a := hashf(ix, iy)
	b := hashf(ix+1, iy)
	c := hashf(ix, iy+1)
	d := hashf(ix+1, iy+1)

	m0 := mixf(a, b, ux)
	m1 := mixf(c, d, ux)
	return mixf(m0, m1, uy)
}

// hashf is a simple 2D hash function
func hashf(x, y float64) float64 {
	v := math.Sin(x*12.9898+y*58.233) * 13758.5453
	return v - math.Floor(v)
}

func bubblePalette(a float64) (r, g, b float64) {
	// Fixed accent color (ANSI 63 ≈ RGB 95,95,255) normalized to 0-1
	_ = a
	return 95.0 / 255.0, 95.0 / 255.0, 1.0
	// Original rainbow cycling:
	// r = 0.5 * (1 + math.Sin(a+0))
	// g = 0.5 * (1 + math.Sin(a+1))
	// b = 0.5 * (1 + math.Sin(a+2))
	// return
}

func mixf(a, b, t float64) float64 {
	return a*(1-t) + b*t
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
