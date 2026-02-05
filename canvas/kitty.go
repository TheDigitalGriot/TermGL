// Package canvas provides the terminal framebuffer abstraction.
package canvas

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
)

// KittyBackend renders images using the Kitty graphics protocol.
// Supported by: Kitty, WezTerm, Ghostty
type KittyBackend struct {
	width     int
	height    int
	pixels    []color.RGBA
	writer    io.Writer
	imageID   uint32
	placement uint32
}

// NewKittyBackend creates a new Kitty graphics backend.
func NewKittyBackend(width, height int) *KittyBackend {
	return &KittyBackend{
		width:   width,
		height:  height,
		pixels:  make([]color.RGBA, width*height),
		writer:  os.Stdout,
		imageID: 1,
	}
}

// SetWriter sets the output writer (default is os.Stdout).
func (k *KittyBackend) SetWriter(w io.Writer) {
	k.writer = w
}

// Width returns the pixel width.
func (k *KittyBackend) Width() int {
	return k.width
}

// Height returns the pixel height.
func (k *KittyBackend) Height() int {
	return k.height
}

// SetPixel sets a pixel at the given position.
func (k *KittyBackend) SetPixel(x, y int, c color.RGBA) {
	if x < 0 || x >= k.width || y < 0 || y >= k.height {
		return
	}
	k.pixels[y*k.width+x] = c
}

// GetPixel returns the pixel at the given position.
func (k *KittyBackend) GetPixel(x, y int) color.RGBA {
	if x < 0 || x >= k.width || y < 0 || y >= k.height {
		return color.RGBA{}
	}
	return k.pixels[y*k.width+x]
}

// Clear clears all pixels to transparent.
func (k *KittyBackend) Clear() {
	for i := range k.pixels {
		k.pixels[i] = color.RGBA{}
	}
}

// ClearColor clears all pixels to a specific color.
func (k *KittyBackend) ClearColor(c color.RGBA) {
	for i := range k.pixels {
		k.pixels[i] = c
	}
}

// Resize resizes the pixel buffer.
func (k *KittyBackend) Resize(width, height int) {
	if width == k.width && height == k.height {
		return
	}
	k.width = width
	k.height = height
	k.pixels = make([]color.RGBA, width*height)
}

// ToImage converts the pixel buffer to an image.Image.
func (k *KittyBackend) ToImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, k.width, k.height))
	for y := 0; y < k.height; y++ {
		for x := 0; x < k.width; x++ {
			img.Set(x, y, k.pixels[y*k.width+x])
		}
	}
	return img
}

// Flush sends the image to the terminal using Kitty graphics protocol.
func (k *KittyBackend) Flush() error {
	return k.FlushAt(0, 0)
}

// FlushAt sends the image to the terminal at a specific position.
func (k *KittyBackend) FlushAt(x, y int) error {
	img := k.ToImage()

	// Encode image as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Send using Kitty protocol
	// Format: ESC_G<control data>;<payload>ESC\
	// Control data: a=T (transmit), f=100 (PNG), i=<id>, ...

	k.placement++

	// Delete previous placement if any
	if k.placement > 1 {
		fmt.Fprintf(k.writer, "\x1b_Ga=d,d=i,i=%d\x1b\\", k.imageID)
	}

	// Send image in chunks (max 4096 bytes per chunk)
	const chunkSize = 4096
	chunks := (len(encoded) + chunkSize - 1) / chunkSize

	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[start:end]

		if i == 0 {
			// First chunk includes control data
			more := 0
			if i < chunks-1 {
				more = 1
			}
			fmt.Fprintf(k.writer, "\x1b_Ga=T,f=100,i=%d,p=%d,q=2,m=%d,c=%d,r=%d;%s\x1b\\",
				k.imageID, k.placement, more, x, y, chunk)
		} else {
			// Continuation chunks
			more := 0
			if i < chunks-1 {
				more = 1
			}
			fmt.Fprintf(k.writer, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}

	return nil
}

// Delete removes all images from the terminal.
func (k *KittyBackend) Delete() {
	fmt.Fprintf(k.writer, "\x1b_Ga=d,d=A\x1b\\")
}

// DeleteByID removes a specific image by ID.
func (k *KittyBackend) DeleteByID(id uint32) {
	fmt.Fprintf(k.writer, "\x1b_Ga=d,d=i,i=%d\x1b\\", id)
}

// DrawLine draws a line on the pixel buffer.
func (k *KittyBackend) DrawLine(x1, y1, x2, y2 int, c color.RGBA) {
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
		k.SetPixel(x1, y1, c)
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
func (k *KittyBackend) DrawRect(x1, y1, x2, y2 int, c color.RGBA) {
	k.DrawLine(x1, y1, x2, y1, c)
	k.DrawLine(x2, y1, x2, y2, c)
	k.DrawLine(x2, y2, x1, y2, c)
	k.DrawLine(x1, y2, x1, y1, c)
}

// FillRect fills a rectangle.
func (k *KittyBackend) FillRect(x1, y1, x2, y2 int, c color.RGBA) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			k.SetPixel(x, y, c)
		}
	}
}

// DrawCircle draws a circle outline.
func (k *KittyBackend) DrawCircle(cx, cy, r int, c color.RGBA) {
	x := 0
	y := r
	d := 3 - 2*r

	k.setCirclePoints(cx, cy, x, y, c)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		k.setCirclePoints(cx, cy, x, y, c)
	}
}

func (k *KittyBackend) setCirclePoints(cx, cy, x, y int, c color.RGBA) {
	k.SetPixel(cx+x, cy+y, c)
	k.SetPixel(cx-x, cy+y, c)
	k.SetPixel(cx+x, cy-y, c)
	k.SetPixel(cx-x, cy-y, c)
	k.SetPixel(cx+y, cy+x, c)
	k.SetPixel(cx-y, cy+x, c)
	k.SetPixel(cx+y, cy-x, c)
	k.SetPixel(cx-y, cy-x, c)
}

// FillCircle fills a circle.
func (k *KittyBackend) FillCircle(cx, cy, r int, c color.RGBA) {
	x := 0
	y := r
	d := 3 - 2*r

	k.fillCircleLines(cx, cy, x, y, c)

	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		k.fillCircleLines(cx, cy, x, y, c)
	}
}

func (k *KittyBackend) fillCircleLines(cx, cy, x, y int, c color.RGBA) {
	for px := cx - x; px <= cx+x; px++ {
		k.SetPixel(px, cy+y, c)
		k.SetPixel(px, cy-y, c)
	}
	for px := cx - y; px <= cx+y; px++ {
		k.SetPixel(px, cy+x, c)
		k.SetPixel(px, cy-x, c)
	}
}

// IsKittySupported checks if the terminal supports Kitty graphics.
func IsKittySupported() bool {
	// Check TERM and TERM_PROGRAM environment variables
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Known Kitty protocol supporters
	kittyTerms := []string{"xterm-kitty", "kitty", "wezterm", "ghostty"}
	for _, kt := range kittyTerms {
		if term == kt || termProgram == kt {
			return true
		}
	}

	// Check for Kitty environment variable
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}

	return false
}
