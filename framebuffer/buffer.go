// Package framebuffer provides an image.RGBA-based pixel buffer for terminal rendering.
// Applications draw to this buffer using standard Go image operations, and the
// compositor handles encoding to the best available terminal format.
package framebuffer

import (
	"image"
	"image/color"
)

// Framebuffer is the application-facing drawing surface.
// Draw to Pixels using standard Go image operations; the compositor reads from it.
type Framebuffer struct {
	// Pixels is the primary RGBA pixel buffer at virtual resolution.
	// Applications draw directly to this buffer.
	Pixels *image.RGBA

	// Width is the virtual pixel width (same as Pixels.Bounds().Dx())
	Width int

	// Height is the virtual pixel height (same as Pixels.Bounds().Dy())
	Height int

	// Dirty tracks the region that has changed since last render.
	// Set to full bounds when unsure; the encoder will optimize.
	Dirty image.Rectangle

	// prev holds the previous frame for delta diffing
	prev *image.RGBA

	// AutoSize causes the framebuffer to resize when the terminal changes
	AutoSize bool
}

// Option configures a Framebuffer
type Option func(*Framebuffer)

// WithFixedSize sets explicit pixel dimensions
func WithFixedSize(width, height int) Option {
	return func(fb *Framebuffer) {
		fb.Width = width
		fb.Height = height
		fb.AutoSize = false
	}
}

// WithAutoSize enables automatic sizing to fill the terminal.
// The actual resolution depends on the encoding level.
func WithAutoSize() Option {
	return func(fb *Framebuffer) {
		fb.AutoSize = true
	}
}

// New creates a new Framebuffer with the given options.
// If no size is specified and AutoSize is not set, defaults to 160x80.
func New(opts ...Option) *Framebuffer {
	fb := &Framebuffer{
		Width:    160,
		Height:   80,
		AutoSize: false,
	}

	for _, opt := range opts {
		opt(fb)
	}

	fb.allocate()
	return fb
}

// allocate creates the pixel buffers at the current dimensions
func (fb *Framebuffer) allocate() {
	bounds := image.Rect(0, 0, fb.Width, fb.Height)
	fb.Pixels = image.NewRGBA(bounds)
	fb.prev = image.NewRGBA(bounds)
	fb.Dirty = bounds
}

// Resize changes the framebuffer dimensions.
// Existing content is preserved where it fits.
func (fb *Framebuffer) Resize(width, height int) {
	if width == fb.Width && height == fb.Height {
		return
	}

	oldPixels := fb.Pixels
	fb.Width = width
	fb.Height = height
	fb.allocate()

	// Copy old content that fits
	if oldPixels != nil {
		copyBounds := fb.Pixels.Bounds().Intersect(oldPixels.Bounds())
		for y := copyBounds.Min.Y; y < copyBounds.Max.Y; y++ {
			for x := copyBounds.Min.X; x < copyBounds.Max.X; x++ {
				fb.Pixels.Set(x, y, oldPixels.At(x, y))
			}
		}
	}

	// Mark entire frame dirty after resize
	fb.MarkAllDirty()
}

// Clear fills the entire framebuffer with the given color
func (fb *Framebuffer) Clear(c color.Color) {
	bounds := fb.Pixels.Bounds()
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			fb.Pixels.SetRGBA(x, y, rgba)
		}
	}
	fb.MarkAllDirty()
}

// ClearToBlack fills the framebuffer with black (efficient path)
func (fb *Framebuffer) ClearToBlack() {
	for i := range fb.Pixels.Pix {
		fb.Pixels.Pix[i] = 0
	}
	fb.MarkAllDirty()
}

// MarkAllDirty marks the entire framebuffer as needing redraw
func (fb *Framebuffer) MarkAllDirty() {
	fb.Dirty = fb.Pixels.Bounds()
}

// MarkClean marks the framebuffer as fully rendered.
// Call this after the compositor has processed the frame.
func (fb *Framebuffer) MarkClean() {
	fb.Dirty = image.Rectangle{}
}

// MarkRegionDirty expands the dirty region to include the given rectangle
func (fb *Framebuffer) MarkRegionDirty(r image.Rectangle) {
	if fb.Dirty.Empty() {
		fb.Dirty = r
	} else {
		fb.Dirty = fb.Dirty.Union(r)
	}
}

// SwapPrevious swaps current and previous frame buffers.
// Call this after rendering to prepare for delta detection.
func (fb *Framebuffer) SwapPrevious() {
	// Copy current to prev for next frame's delta comparison
	copy(fb.prev.Pix, fb.Pixels.Pix)
}

// Previous returns the previous frame for delta comparison
func (fb *Framebuffer) Previous() *image.RGBA {
	return fb.prev
}

// SetPixel sets a single pixel (convenience method)
func (fb *Framebuffer) SetPixel(x, y int, c color.RGBA) {
	if x >= 0 && x < fb.Width && y >= 0 && y < fb.Height {
		fb.Pixels.SetRGBA(x, y, c)
	}
}

// GetPixel gets a single pixel (convenience method)
func (fb *Framebuffer) GetPixel(x, y int) color.RGBA {
	if x >= 0 && x < fb.Width && y >= 0 && y < fb.Height {
		return fb.Pixels.RGBAAt(x, y)
	}
	return color.RGBA{}
}

// Bounds returns the framebuffer bounds
func (fb *Framebuffer) Bounds() image.Rectangle {
	return fb.Pixels.Bounds()
}
