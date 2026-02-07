// Package compose provides the adaptive compositor that selects the best
// encoder for the current terminal and handles output.
package compose

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/termgl/detect"
	"github.com/charmbracelet/termgl/encode"
	"github.com/charmbracelet/termgl/framebuffer"
)

// FlushStrategy controls how output is written to the terminal
type FlushStrategy int

const (
	// FlushImmediate writes directly to stdout (may cause tearing)
	FlushImmediate FlushStrategy = iota

	// FlushBuffered collects all output before writing (reduces tearing)
	FlushBuffered

	// FlushSynchronized uses DEC synchronized output mode (no tearing)
	// Supported by: Windows Terminal, Kitty, WezTerm, foot
	FlushSynchronized
)

// Compositor orchestrates encoding and output to the terminal
type Compositor struct {
	caps    *detect.TerminalCaps
	encoder encode.Encoder
	output  io.Writer
	flush   FlushStrategy

	// Screen management
	altScreen bool
	cursorHidden bool

	// Frame tracking
	frameCount uint64
}

// Option configures a Compositor
type Option func(*Compositor)

// WithOutput sets the output writer (default: os.Stdout)
func WithOutput(w io.Writer) Option {
	return func(c *Compositor) {
		c.output = w
	}
}

// WithFlushStrategy sets the flush strategy
func WithFlushStrategy(s FlushStrategy) Option {
	return func(c *Compositor) {
		c.flush = s
	}
}

// New creates a Compositor for the given terminal capabilities
func New(caps *detect.TerminalCaps, opts ...Option) *Compositor {
	c := &Compositor{
		caps:   caps,
		output: os.Stdout,
		flush:  FlushBuffered,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Auto-select flush strategy based on capabilities
	if caps.HasSyncOutput && c.flush == FlushBuffered {
		c.flush = FlushSynchronized
	}

	// Create encoder for best available level
	c.createEncoder(caps.BestLevel)

	return c
}

// createEncoder creates the appropriate encoder for the given level
func (c *Compositor) createEncoder(level detect.EncoderLevel) {
	switch level {
	case detect.LevelHalfBlock, detect.LevelQuadrant, detect.LevelBraille:
		// All use half-block for now (quadrant and braille to be added)
		c.encoder = encode.NewHalfBlockEncoder(c.caps.GridSize.X, c.caps.GridSize.Y)
	default:
		// Fallback to half-block
		c.encoder = encode.NewHalfBlockEncoder(c.caps.GridSize.X, c.caps.GridSize.Y)
	}
}

// EnterAltScreen switches to the alternate screen buffer
func (c *Compositor) EnterAltScreen() {
	if c.altScreen {
		return
	}
	c.write([]byte("\x1b[?1049h")) // Enter alt screen
	c.altScreen = true
}

// ExitAltScreen returns to the main screen buffer
func (c *Compositor) ExitAltScreen() {
	if !c.altScreen {
		return
	}
	c.write([]byte("\x1b[?1049l")) // Exit alt screen
	c.altScreen = false
}

// HideCursor hides the terminal cursor
func (c *Compositor) HideCursor() {
	if c.cursorHidden {
		return
	}
	c.write([]byte("\x1b[?25l")) // Hide cursor
	c.cursorHidden = true
}

// ShowCursor shows the terminal cursor
func (c *Compositor) ShowCursor() {
	if !c.cursorHidden {
		return
	}
	c.write([]byte("\x1b[?25h")) // Show cursor
	c.cursorHidden = false
}

// ClearScreen clears the terminal screen
func (c *Compositor) ClearScreen() {
	c.write([]byte("\x1b[2J")) // Clear entire screen
	c.write([]byte("\x1b[H"))  // Move cursor to home
}

// Render encodes the framebuffer and outputs to the terminal
func (c *Compositor) Render(fb *framebuffer.Framebuffer) {
	data := c.encoder.Encode(fb)
	if len(data) == 0 {
		return // Nothing changed
	}

	c.flushFrame(data)
	fb.SwapPrevious()
	c.frameCount++
}

// RenderFull forces a complete redraw without delta optimization
func (c *Compositor) RenderFull(fb *framebuffer.Framebuffer) {
	c.encoder.Reset()

	// For half-block encoder, use full frame method
	if hb, ok := c.encoder.(*encode.HalfBlockEncoder); ok {
		data := hb.EncodeFullFrame(fb)
		c.flushFrame(data)
	} else {
		data := c.encoder.Encode(fb)
		c.flushFrame(data)
	}

	fb.SwapPrevious()
	c.frameCount++
}

// flushFrame writes frame data with the configured flush strategy
func (c *Compositor) flushFrame(data []byte) {
	switch c.flush {
	case FlushSynchronized:
		// DEC synchronized output mode
		c.write([]byte("\x1b[?2026h")) // Begin sync
		c.write(data)
		c.write([]byte("\x1b[?2026l")) // End sync
	default:
		c.write(data)
	}
}

// write sends bytes to the output
func (c *Compositor) write(data []byte) {
	c.output.Write(data)
}

// Reset clears encoder state and forces full redraw on next render
func (c *Compositor) Reset() {
	c.encoder.Reset()
}

// Cleanup restores terminal state (call before exit)
func (c *Compositor) Cleanup() {
	c.ShowCursor()
	c.ExitAltScreen()
	c.write([]byte("\x1b[0m")) // Reset colors
}

// Caps returns the detected terminal capabilities
func (c *Compositor) Caps() *detect.TerminalCaps {
	return c.caps
}

// FrameCount returns the number of frames rendered
func (c *Compositor) FrameCount() uint64 {
	return c.frameCount
}

// VirtualSize returns the virtual pixel resolution for the current encoder
func (c *Compositor) VirtualSize() (width, height int) {
	res := c.caps.VirtualResolution(c.encoder.Level())
	return res.X, res.Y
}

// GridSize returns the terminal grid size in cells
func (c *Compositor) GridSize() (cols, rows int) {
	return c.caps.GridSize.X, c.caps.GridSize.Y
}

// PrintDebugInfo writes terminal capability info to the given writer
func (c *Compositor) PrintDebugInfo(w io.Writer) {
	fmt.Fprintf(w, "Terminal: %s\n", c.caps.Terminal)
	fmt.Fprintf(w, "OS: %s\n", c.caps.OS)
	fmt.Fprintf(w, "Color Depth: %d\n", c.caps.ColorDepth)
	fmt.Fprintf(w, "Grid Size: %dx%d\n", c.caps.GridSize.X, c.caps.GridSize.Y)
	fmt.Fprintf(w, "Best Level: %s\n", c.caps.BestLevel.String())
	fmt.Fprintf(w, "Has Unicode: %v\n", c.caps.HasUnicode)
	fmt.Fprintf(w, "Has Braille: %v\n", c.caps.HasBraille)
	fmt.Fprintf(w, "Has Sixel: %v\n", c.caps.HasSixel)
	fmt.Fprintf(w, "Has Kitty: %v\n", c.caps.HasKitty)
	fmt.Fprintf(w, "Has Sync Output: %v\n", c.caps.HasSyncOutput)

	vw, vh := c.VirtualSize()
	fmt.Fprintf(w, "Virtual Resolution: %dx%d\n", vw, vh)
}
