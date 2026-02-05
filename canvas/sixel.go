// Package canvas provides the terminal framebuffer abstraction.
package canvas

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"sort"
)

// SixelBackend renders images using the Sixel graphics protocol.
// Supported by: xterm, foot, WezTerm, Konsole, VS Code terminal, mlterm, etc.
type SixelBackend struct {
	width   int
	height  int
	pixels  []color.RGBA
	writer  io.Writer
	palette []color.RGBA
}

// NewSixelBackend creates a new Sixel graphics backend.
func NewSixelBackend(width, height int) *SixelBackend {
	return &SixelBackend{
		width:   width,
		height:  height,
		pixels:  make([]color.RGBA, width*height),
		writer:  os.Stdout,
		palette: make([]color.RGBA, 0, 256),
	}
}

// SetWriter sets the output writer (default is os.Stdout).
func (s *SixelBackend) SetWriter(w io.Writer) {
	s.writer = w
}

// Width returns the pixel width.
func (s *SixelBackend) Width() int {
	return s.width
}

// Height returns the pixel height.
func (s *SixelBackend) Height() int {
	return s.height
}

// SetPixel sets a pixel at the given position.
func (s *SixelBackend) SetPixel(x, y int, c color.RGBA) {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return
	}
	s.pixels[y*s.width+x] = c
}

// GetPixel returns the pixel at the given position.
func (s *SixelBackend) GetPixel(x, y int) color.RGBA {
	if x < 0 || x >= s.width || y < 0 || y >= s.height {
		return color.RGBA{}
	}
	return s.pixels[y*s.width+x]
}

// Clear clears all pixels to transparent.
func (s *SixelBackend) Clear() {
	for i := range s.pixels {
		s.pixels[i] = color.RGBA{}
	}
}

// ClearColor clears all pixels to a specific color.
func (s *SixelBackend) ClearColor(c color.RGBA) {
	for i := range s.pixels {
		s.pixels[i] = c
	}
}

// Resize resizes the pixel buffer.
func (s *SixelBackend) Resize(width, height int) {
	if width == s.width && height == s.height {
		return
	}
	s.width = width
	s.height = height
	s.pixels = make([]color.RGBA, width*height)
}

// ToImage converts the pixel buffer to an image.Image.
func (s *SixelBackend) ToImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, s.width, s.height))
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			img.Set(x, y, s.pixels[y*s.width+x])
		}
	}
	return img
}

// Flush sends the image to the terminal using Sixel protocol.
func (s *SixelBackend) Flush() error {
	// Build color palette from image
	s.buildPalette()

	// Generate Sixel data
	var buf bytes.Buffer

	// Sixel start sequence
	// DCS P1 ; P2 ; P3 q
	// P1=7 (pixel ratio), P2=1 (background), P3=0 (horizontal grid size)
	buf.WriteString("\x1bPq")

	// Define color palette
	// # Pc ; Pu ; Px ; Py ; Pz
	// Pc = color number, Pu = color coordinate system (2 = RGB)
	// Px, Py, Pz = R, G, B (0-100 scale)
	for i, c := range s.palette {
		r := int(c.R) * 100 / 255
		g := int(c.G) * 100 / 255
		b := int(c.B) * 100 / 255
		fmt.Fprintf(&buf, "#%d;2;%d;%d;%d", i, r, g, b)
	}

	// Generate sixel data
	// Sixels are 6 pixels tall, so we process in bands of 6 rows
	for band := 0; band*6 < s.height; band++ {
		if band > 0 {
			buf.WriteByte('-') // Graphics new line
		}

		// For each color in palette
		for colorIdx := range s.palette {
			// Select color
			fmt.Fprintf(&buf, "#%d", colorIdx)

			// Build sixel data for this color in this band
			var runLength int
			var lastSixel byte

			for x := 0; x < s.width; x++ {
				sixel := s.buildSixel(x, band*6, colorIdx)

				if sixel == lastSixel && runLength > 0 {
					runLength++
				} else {
					// Flush previous run
					if runLength > 0 {
						s.writeSixelRun(&buf, lastSixel, runLength)
					}
					lastSixel = sixel
					runLength = 1
				}
			}

			// Flush final run
			if runLength > 0 {
				s.writeSixelRun(&buf, lastSixel, runLength)
			}

			buf.WriteByte('$') // Graphics carriage return
		}
	}

	// Sixel end sequence
	buf.WriteString("\x1b\\")

	_, err := s.writer.Write(buf.Bytes())
	return err
}

// buildPalette extracts a color palette from the image.
// Uses simple color quantization (max 256 colors).
func (s *SixelBackend) buildPalette() {
	colorCounts := make(map[color.RGBA]int)

	for _, c := range s.pixels {
		if c.A == 0 {
			continue // Skip transparent
		}
		// Quantize to reduce colors
		qc := color.RGBA{
			R: (c.R / 8) * 8,
			G: (c.G / 8) * 8,
			B: (c.B / 8) * 8,
			A: 255,
		}
		colorCounts[qc]++
	}

	// Sort by frequency
	type colorCount struct {
		color color.RGBA
		count int
	}
	var counts []colorCount
	for c, n := range colorCounts {
		counts = append(counts, colorCount{c, n})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	// Take top 256 colors
	s.palette = s.palette[:0]
	for i := 0; i < len(counts) && i < 256; i++ {
		s.palette = append(s.palette, counts[i].color)
	}

	// Ensure we have at least one color
	if len(s.palette) == 0 {
		s.palette = append(s.palette, color.RGBA{0, 0, 0, 255})
	}
}

// findNearestColor finds the palette index for a color.
func (s *SixelBackend) findNearestColor(c color.RGBA) int {
	if c.A == 0 {
		return -1 // Transparent
	}

	qc := color.RGBA{
		R: (c.R / 8) * 8,
		G: (c.G / 8) * 8,
		B: (c.B / 8) * 8,
		A: 255,
	}

	// First try exact match
	for i, pc := range s.palette {
		if pc == qc {
			return i
		}
	}

	// Find nearest color
	bestIdx := 0
	bestDist := 256 * 256 * 3

	for i, pc := range s.palette {
		dr := int(pc.R) - int(qc.R)
		dg := int(pc.G) - int(qc.G)
		db := int(pc.B) - int(qc.B)
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}

// buildSixel creates a sixel byte for a column at the given band.
func (s *SixelBackend) buildSixel(x, bandY, colorIdx int) byte {
	var sixel byte

	for bit := 0; bit < 6; bit++ {
		y := bandY + bit
		if y >= s.height {
			break
		}

		c := s.pixels[y*s.width+x]
		ci := s.findNearestColor(c)
		if ci == colorIdx {
			sixel |= 1 << bit
		}
	}

	return sixel + 63 // Sixel characters start at '?'
}

// writeSixelRun writes a run-length encoded sixel sequence.
func (s *SixelBackend) writeSixelRun(buf *bytes.Buffer, sixel byte, length int) {
	if length == 1 {
		buf.WriteByte(sixel)
	} else if length == 2 {
		buf.WriteByte(sixel)
		buf.WriteByte(sixel)
	} else {
		fmt.Fprintf(buf, "!%d%c", length, sixel)
	}
}

// DrawLine draws a line on the pixel buffer.
func (s *SixelBackend) DrawLine(x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		s.SetPixel(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// DrawRect draws a rectangle outline.
func (s *SixelBackend) DrawRect(x1, y1, x2, y2 int, c color.RGBA) {
	s.DrawLine(x1, y1, x2, y1, c)
	s.DrawLine(x2, y1, x2, y2, c)
	s.DrawLine(x2, y2, x1, y2, c)
	s.DrawLine(x1, y2, x1, y1, c)
}

// FillRect fills a rectangle.
func (s *SixelBackend) FillRect(x1, y1, x2, y2 int, c color.RGBA) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			s.SetPixel(x, y, c)
		}
	}
}

// IsSixelSupported checks if the terminal supports Sixel graphics.
func IsSixelSupported() bool {
	// Check TERM environment variable
	term := os.Getenv("TERM")

	// Known Sixel supporting terminals
	sixelTerms := []string{
		"xterm", "xterm-256color", "xterm-direct",
		"foot", "foot-direct",
		"mlterm", "mlterm-256color",
		"yaft-256color",
	}

	for _, st := range sixelTerms {
		if term == st {
			return true
		}
	}

	// Check for WezTerm (supports both Kitty and Sixel)
	if os.Getenv("TERM_PROGRAM") == "WezTerm" {
		return true
	}

	// Check for SIXEL capability in terminfo (would require terminfo parsing)
	// For now, we'll be conservative

	return false
}

// PixelBackendType indicates which pixel backend to use.
type PixelBackendType int

const (
	PixelBackendNone PixelBackendType = iota
	PixelBackendKitty
	PixelBackendSixel
)

// DetectPixelBackend auto-detects the best available pixel backend.
func DetectPixelBackend() PixelBackendType {
	if IsKittySupported() {
		return PixelBackendKitty
	}
	if IsSixelSupported() {
		return PixelBackendSixel
	}
	return PixelBackendNone
}
