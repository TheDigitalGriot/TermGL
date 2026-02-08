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
	"github.com/fogleman/fauxgl"
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

	// Set up camera
	scene.Camera.SetPosition(0, 0, 3)
	scene.Camera.SetTarget(0, 0, 0)

	// Add a directional light
	light := render.NewDirectionalLight(0.5, 0.5, 1, fauxgl.Color{R: 1, G: 1, B: 1, A: 1}, 0.8)
	scene.Lights = append(scene.Lights, light)

	// Set ambient light
	scene.Ambient = fauxgl.Color{R: 0.15, G: 0.15, B: 0.2, A: 1}

	// Create renderer at Sixel resolution
	width, height := 320, 240
	renderer := render.NewRenderer(width, height)

	// Create Sixel pipeline
	// Use OctreeQuantizer with Bayer 4x4 dithering (best for animation)
	octreeQuant := tier1.NewOctreeQuantizer(tier1.DitherOrdered4x4, tier1.ColorSpaceRGB)

	// Wrap in StablePaletteQuantizer for flicker-free animation
	stableQuant := tier1.NewStablePaletteQuantizer(0.3, 32, octreeQuant)

	// Create SixelOutput
	sixelOut := tier1.NewSixelOutput(stableQuant, 256, true)

	// Initialize with default terminal caps
	caps := render.TerminalCaps{
		Sixel:          true,
		SixelMaxColors: 256,
		CellWidth:      8,
		CellHeight:     16,
		Width:          80,
		Height:         24,
	}
	if err := sixelOut.Init(caps); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing SixelOutput: %v\n", err)
		os.Exit(1)
	}

	// Hide cursor
	fmt.Print("\x1b[?25l")
	// Clear screen
	fmt.Print("\x1b[2J\x1b[H")

	defer func() {
		// Show cursor
		fmt.Print("\x1b[?25h")
		// Reset colors
		fmt.Print("\x1b[0m")
	}()

	// Animation loop
	const targetFPS = 30
	frameDuration := time.Second / targetFPS
	frameCount := 0
	startTime := time.Now()
	lastFPSReport := startTime

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

		// Encode to Sixel with cursor positioning
		output := sixelOut.EncodeWithCursor(img, 1, 1)

		// Write to stdout
		fmt.Print(output)

		frameCount++

		// Report FPS every 2 seconds
		now := time.Now()
		if now.Sub(lastFPSReport) >= 2*time.Second {
			elapsed := now.Sub(startTime).Seconds()
			fps := float64(frameCount) / elapsed
			// Print FPS below the image
			imgRows := (height + 15) / 16 // Approximate rows based on cell height
			fmt.Printf("\x1b[%d;1H\x1b[K FPS: %.1f  Frame: %d  Sixel size: %d bytes",
				imgRows+1, fps, frameCount, len(output))
			lastFPSReport = now
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
	imgRows := (height + 15) / 16
	fmt.Printf("\x1b[%d;1H\x1b[K Final: %d frames in %.1fs = %.1f FPS\n",
		imgRows+2, frameCount, elapsed, fps)
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
