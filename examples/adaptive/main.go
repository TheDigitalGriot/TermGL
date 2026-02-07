// Example: Adaptive Framebuffer Demo
//
// Demonstrates the adaptive framebuffer system with a bouncing gradient circle.
// Works on PowerShell, Windows Terminal, and any terminal with 24-bit color.
//
// Run with: go run ./examples/adaptive
// Press Ctrl+C to exit.
package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/termgl/compose"
	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/framebuffer"
)

func main() {
	// 1. Detect terminal capabilities
	caps := detect.Capabilities()

	fmt.Printf("Adaptive Framebuffer Demo\n")
	fmt.Printf("Terminal: %s\n", caps.Terminal)
	fmt.Printf("Grid: %dx%d cells\n", caps.GridSize.X, caps.GridSize.Y)
	fmt.Printf("Best Level: %s\n", caps.BestLevel.String())
	fmt.Printf("\nStarting in 2 seconds... (Ctrl+C to exit)\n")
	time.Sleep(2 * time.Second)

	// 2. Create compositor
	comp := compose.New(caps)

	// 3. Get virtual pixel resolution
	vw, vh := comp.VirtualSize()
	fmt.Printf("Virtual resolution: %dx%d pixels\n", vw, vh)
	time.Sleep(1 * time.Second)

	// 4. Create framebuffer at virtual resolution
	fb := framebuffer.New(framebuffer.WithFixedSize(vw, vh))

	// 5. Setup terminal
	comp.EnterAltScreen()
	comp.HideCursor()
	comp.ClearScreen()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		<-sigChan
		done <- true
	}()

	// 6. Animation loop
	t := 0.0
	frameStart := time.Now()
	frames := 0

	// Circle properties
	radius := float64(min(vw, vh)) * 0.15
	centerX := float64(vw) / 2
	centerY := float64(vh) / 2

	for {
		select {
		case <-done:
			goto cleanup
		default:
		}

		// Clear to dark background
		clearColor := color.RGBA{R: 10, G: 10, B: 20, A: 255}
		fb.Clear(clearColor)

		// Calculate bouncing circle position
		cx := centerX + math.Sin(t)*float64(vw)*0.35
		cy := centerY + math.Cos(t*0.7)*float64(vh)*0.35

		// Draw gradient circle
		drawGradientCircle(fb, cx, cy, radius, t)

		// Draw a second smaller circle orbiting the first
		orbitRadius := radius * 2.5
		cx2 := cx + math.Cos(t*3)*orbitRadius
		cy2 := cy + math.Sin(t*3)*orbitRadius
		drawGradientCircle(fb, cx2, cy2, radius*0.4, t+math.Pi)

		// Render frame
		if frames == 0 {
			comp.RenderFull(fb)
		} else {
			comp.Render(fb)
		}

		// Frame timing
		t += 0.05
		frames++
		time.Sleep(33 * time.Millisecond) // ~30 FPS

		// Print FPS every second
		if frames%30 == 0 {
			elapsed := time.Since(frameStart).Seconds()
			fps := float64(frames) / elapsed
			// Move cursor to top-left and print FPS (on top of rendered content)
			fmt.Printf("\x1b[1;1H\x1b[97mFPS: %.1f | Frames: %d\x1b[0m", fps, frames)
		}
	}

cleanup:
	// 7. Cleanup
	comp.Cleanup()
	fmt.Printf("\n\nRendered %d frames\n", frames)
	elapsed := time.Since(frameStart).Seconds()
	fmt.Printf("Average FPS: %.1f\n", float64(frames)/elapsed)
}

// drawGradientCircle draws a circle with radial gradient coloring
func drawGradientCircle(fb *framebuffer.Framebuffer, cx, cy, radius, phase float64) {
	// Calculate bounding box
	minX := int(math.Max(0, cx-radius-1))
	maxX := int(math.Min(float64(fb.Width-1), cx+radius+1))
	minY := int(math.Max(0, cy-radius-1))
	maxY := int(math.Min(float64(fb.Height-1), cy+radius+1))

	radiusSq := radius * radius

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			distSq := dx*dx + dy*dy

			if distSq <= radiusSq {
				// Calculate intensity based on distance from center
				dist := math.Sqrt(distSq)
				intensity := 1.0 - dist/radius

				// Smooth edge with antialiasing
				edgeDist := radius - dist
				if edgeDist < 1.0 && edgeDist > 0 {
					intensity *= edgeDist
				}

				// Calculate angle for color variation
				angle := math.Atan2(dy, dx)

				// Create animated gradient colors
				r := uint8(clamp(255*intensity*(0.5+0.5*math.Sin(angle+phase)), 0, 255))
				g := uint8(clamp(255*intensity*(0.5+0.5*math.Sin(angle+phase+2.094)), 0, 255))
				b := uint8(clamp(255*intensity*(0.5+0.5*math.Sin(angle+phase+4.188)), 0, 255))

				fb.SetPixel(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}
}

func clamp(v, minVal, maxVal float64) float64 {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
