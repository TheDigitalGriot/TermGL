package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
	"github.com/charmbracelet/termgl/tier2"
	"github.com/fogleman/fauxgl"
	"golang.org/x/term"
)

func main() {
	// Get terminal size
	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth, termHeight = 80, 24
	}

	// Load Suzanne mesh
	mesh, err := render.LoadOBJ("models/suzanne_blender_monkey.obj")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading mesh: %v\n", err)
		os.Exit(1)
	}
	mesh.BiUnitCube()

	// Create scene
	scene := render.NewScene()
	meshNode := render.NewMeshNode(mesh)
	scene.Root.AddChild(meshNode)

	scene.Camera.SetPosition(0, 0, 3)
	scene.Camera.SetTarget(0, 0, 0)

	light := render.NewDirectionalLight(0.5, 0.5, 1, fauxgl.Color{R: 1, G: 1, B: 1, A: 1}, 0.8)
	scene.Lights = append(scene.Lights, light)
	scene.Ambient = fauxgl.Color{R: 0.12, G: 0.12, B: 0.18, A: 1}

	// Tier 2 pipeline configuration
	blitterMode := 1 // 0=half, 1=quad, 2=sextant, 3=braille
	ditherMode := tier1.DitherOrdered4x4
	useFreqSplit := true
	useEdgeAware := true
	useDelta := true

	// Create blitter based on mode
	var blitter tier2.Blitter
	switch blitterMode {
	case 0:
		blitter = tier2.HalfBlockBlitter{}
	case 1:
		blitter = tier2.QuadrantBlitter{}
	case 2:
		blitter = tier2.SextantBlitter{}
	case 3:
		blitter = tier2.BrailleBlitter{}
	default:
		blitter = tier2.SextantBlitter{}
	}

	// Create ANSI encoder with Tier 2 pipeline
	encoder := tier2.NewANSIOutput(blitter).
		WithDither(ditherMode).
		WithFrequencySplit(useFreqSplit).
		WithEdgeAware(useEdgeAware).
		WithDeltaEncoding(useDelta)

	caps := render.TerminalCaps{
		TrueColor: true,
		Unicode:   render.Unicode13,
		Width:     termWidth,
		Height:    termHeight - 2, // Reserve 2 rows for status
	}
	encoder.Init(caps)

	// Calculate internal resolution based on blitter
	imgWidth, imgHeight := encoder.InternalResolution(caps.Width, caps.Height)

	// Create renderer at sub-cell resolution
	renderer := render.NewRenderer(imgWidth, imgHeight)

	// Hide cursor, clear screen
	fmt.Print("\x1b[?25l\x1b[2J")
	defer fmt.Print("\x1b[?25h\x1b[0m")

	// Animation loop
	const targetFPS = 30
	frameDuration := time.Second / targetFPS
	frameCount := 0
	startTime := time.Now()
	lastFPSReport := startTime

	blitterNames := []string{"Half-Block", "Quadrant", "Sextant", "Braille"}

	for {
		frameStart := time.Now()

		// Rotate
		angle := float64(frameCount) * 0.03
		meshNode.SetRotation(0.2, angle, 0)
		scene.Tick(time.Now())

		// Render frame
		img, aux := renderer.RenderFrameWithAux(scene)

		// Composite bubble background behind the 3D model.
		// Detect background pixels by comparing to the ambient clear color.
		t := time.Since(startTime).Seconds()
		compositeWithBubbleBackground(img, t, scene.Ambient)

		// Encode to ANSI with full Tier 2 pipeline
		var output string
		if useEdgeAware {
			output = encoder.EncodeWithAuxAndCursor(img, aux, 1, 1)
		} else {
			output = encoder.EncodeWithCursor(img, 1, 1)
		}
		fmt.Print(output)

		frameCount++

		// Status line
		now := time.Now()
		if now.Sub(lastFPSReport) >= 2*time.Second {
			elapsed := now.Sub(startTime).Seconds()
			fps := float64(frameCount) / elapsed

			cellW, cellH := blitter.SubCellSize()
			fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K Tier 2 ANSI | %s (%dx%d) | %dx%d | %.1f FPS | Frame %d",
				termHeight,
				blitterNames[blitterMode],
				cellW, cellH,
				imgWidth, imgHeight,
				fps,
				frameCount)

			// Feature toggles display
			fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K FreqSplit:%v EdgeAware:%v Delta:%v Dither:%d",
				termHeight-1,
				useFreqSplit,
				useEdgeAware,
				useDelta,
				ditherMode)

			lastFPSReport = now
		}

		// Frame pacing
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}

		if frameCount >= 300 {
			break
		}
	}

	elapsed := time.Since(startTime).Seconds()
	fps := float64(frameCount) / elapsed
	fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K Final: %d frames in %.1fs = %.1f FPS\n",
		termHeight, frameCount, elapsed, fps)
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
	r = 0.5 * (1 + math.Sin(a+0))
	g = 0.5 * (1 + math.Sin(a+1))
	b = 0.5 * (1 + math.Sin(a+2))
	return
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
