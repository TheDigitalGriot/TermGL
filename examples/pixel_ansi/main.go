package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/termgl/render"
	"github.com/charmbracelet/termgl/tier1"
	"github.com/fogleman/fauxgl"
	"golang.org/x/term"
)

func main() {
	// Get terminal size
	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth, termHeight = 80, 24
	}

	// Half-block: each cell = 1px wide, 2px tall
	imgWidth := termWidth
	imgHeight := (termHeight - 2) * 2 // Reserve 2 rows for status

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

	// Create renderer at half-block resolution
	renderer := render.NewRenderer(imgWidth, imgHeight)

	// ANSI half-block encoder — works in ANY 24-bit terminal
	encoder := &tier1.HalfBlockEncoder{}

	// Hide cursor, clear screen
	fmt.Print("\x1b[?25l\x1b[2J")
	defer fmt.Print("\x1b[?25h\x1b[0m")

	// Animation loop
	const targetFPS = 30
	frameDuration := time.Second / targetFPS
	frameCount := 0
	startTime := time.Now()
	lastFPSReport := startTime

	for {
		frameStart := time.Now()

		// Rotate
		angle := float64(frameCount) * 0.03
		meshNode.SetRotation(0.2, angle, 0)
		scene.Tick(time.Now())

		// Render
		img := renderer.RenderFrame(scene)

		// Encode to ANSI half-blocks with cursor positioning
		output := encoder.EncodeWithCursor(img, 1, 1)
		fmt.Print(output)

		frameCount++

		// FPS report
		now := time.Now()
		if now.Sub(lastFPSReport) >= 2*time.Second {
			elapsed := now.Sub(startTime).Seconds()
			fps := float64(frameCount) / elapsed
			fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K ANSI Half-Block | %dx%d | %.1f FPS | Frame %d",
				termHeight, imgWidth, imgHeight, fps, frameCount)
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
