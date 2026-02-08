package main

import (
	"fmt"
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
	caps := tier1.TerminalCaps{
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
