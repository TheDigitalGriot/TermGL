package main

import (
	"fmt"
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

	// Load prism mesh
	mesh, err := render.LoadOBJ("models/prism-test.obj")
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

	light := render.NewDirectionalLight(0.6, 0.5, 1, fauxgl.Color{R: 0.9, G: 0.92, B: 1.0, A: 1}, 0.85)
	scene.Lights = append(scene.Lights, light)
	fill := render.NewDirectionalLight(-0.4, -0.3, 0.5, fauxgl.Color{R: 1.0, G: 0.85, B: 0.7, A: 1}, 0.3)
	scene.Lights = append(scene.Lights, fill)

	scene.Ambient = fauxgl.Color{R: 0.05, G: 0.04, B: 0.08, A: 1}

	// Tier 2 pipeline
	blitterMode := 1
	ditherMode := tier1.DitherOrdered4x4
	useFreqSplit := true
	useEdgeAware := true
	useDelta := true

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
		blitter = tier2.QuadrantBlitter{}
	}

	encoder := tier2.NewANSIOutput(blitter).
		WithDither(ditherMode).
		WithFrequencySplit(useFreqSplit).
		WithEdgeAware(useEdgeAware).
		WithDeltaEncoding(useDelta)

	caps := render.TerminalCaps{
		TrueColor: true,
		Unicode:   render.Unicode13,
		Width:     termWidth,
		Height:    termHeight - 2,
	}
	encoder.Init(caps)

	imgWidth, imgHeight := encoder.InternalResolution(caps.Width, caps.Height)
	renderer := render.NewRenderer(imgWidth, imgHeight)

	// Hide cursor, clear screen
	fmt.Print("\x1b[?25l\x1b[2J")
	defer fmt.Print("\x1b[?25h\x1b[0m")

	const targetFPS = 30
	frameDuration := time.Second / targetFPS
	frameCount := 0
	startTime := time.Now()
	lastFPSReport := startTime

	blitterNames := []string{"Half-Block", "Quadrant", "Sextant", "Braille"}

	for {
		frameStart := time.Now()

		angle := float64(frameCount) * 0.02
		tilt := 0.3 + 0.15*math.Sin(angle*0.7)
		meshNode.SetRotation(tilt, angle, 0.1*math.Sin(angle*0.5))
		scene.Tick(time.Now())

		img, aux := renderer.RenderFrameWithAux(scene)

		var output string
		if useEdgeAware {
			output = encoder.EncodeWithAuxAndCursor(img, aux, 1, 1)
		} else {
			output = encoder.EncodeWithCursor(img, 1, 1)
		}
		fmt.Print(output)

		frameCount++

		now := time.Now()
		if now.Sub(lastFPSReport) >= 2*time.Second {
			elapsed := now.Sub(startTime).Seconds()
			fps := float64(frameCount) / elapsed

			cellW, cellH := blitter.SubCellSize()
			fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K Tier 2 ANSI | %s (%dx%d) | %dx%d | %.1f FPS | prism-test.obj",
				termHeight,
				blitterNames[blitterMode],
				cellW, cellH,
				imgWidth, imgHeight,
				fps)

			fmt.Printf("\x1b[%d;1H\x1b[0m\x1b[K FreqSplit:%v EdgeAware:%v Delta:%v Dither:%d Frame:%d",
				termHeight-1,
				useFreqSplit,
				useEdgeAware,
				useDelta,
				ditherMode,
				frameCount)

			lastFPSReport = now
		}

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
