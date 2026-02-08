package render

import (
	"github.com/charmbracelet/termgl/detect"
)

// TerminalCaps holds detected terminal capabilities.
// This is the canonical version used by the Encoder interface.
// Architecture doc Section 3.1
type TerminalCaps struct {
	// Pixel graphics protocol support
	Sixel          bool
	SixelMaxColors int
	KittyGraphics  bool
	ITerm2         bool

	// ANSI capabilities
	TrueColor bool

	// Unicode support level (used by Tier 2 blitter selection)
	Unicode UnicodeLevel

	// Terminal dimensions in cells
	Width  int
	Height int

	// Cell dimensions in pixels (for Sixel sizing)
	CellWidth  int
	CellHeight int

	// Synchronized output support (for flicker-free rendering)
	SyncOutput bool

	// Terminal name (for debugging/status display)
	Terminal string
}

// UnicodeLevel indicates the terminal's Unicode character support.
// Mirrors tier2.UnicodeLevel but lives in render/ to avoid circular imports.
type UnicodeLevel int

const (
	UnicodeASCII UnicodeLevel = iota // ASCII only
	UnicodeBMP                       // Half-blocks, quadrants
	Unicode13                        // Sextants, braille (Unicode 13+)
	Unicode16                        // Octants (Unicode 16+)
)

// FromDetect converts the existing detect.TerminalCaps to render.TerminalCaps.
// This bridges the "crawl" detection code to the "walk/run" rendering path.
func FromDetect(d *detect.TerminalCaps) TerminalCaps {
	caps := TerminalCaps{
		Sixel:          d.HasSixel,
		SixelMaxColors: 256, // default; detect doesn't expose this
		KittyGraphics:  d.HasKitty,
		ITerm2:         d.HasITerm,
		TrueColor:      d.Is24Bit(),
		Width:          d.GridSize.X,
		Height:         d.GridSize.Y,
		CellWidth:      d.CellSize.X,
		CellHeight:     d.CellSize.Y,
		SyncOutput:     d.HasSyncOutput,
		Terminal:        d.Terminal,
	}

	// Map detect capabilities to UnicodeLevel
	switch {
	case d.HasBraille:
		caps.Unicode = Unicode13 // Braille implies sextant support
	case d.HasUnicode:
		caps.Unicode = UnicodeBMP
	default:
		caps.Unicode = UnicodeASCII
	}

	// Override SixelMaxColors for known high-color terminals
	if d.Terminal == "wezterm" {
		caps.SixelMaxColors = 1024
	}

	return caps
}

// Detect runs terminal capability detection and returns render.TerminalCaps.
// Convenience wrapper around detect.Capabilities() + FromDetect().
func Detect() TerminalCaps {
	return FromDetect(detect.Capabilities())
}
