package canvas

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color is an alias for lipgloss.Color for convenience.
type Color = lipgloss.Color

// RGB creates a 24-bit true color from RGB values.
func RGB(r, g, b uint8) Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// Gray creates a grayscale color from a single value (0-255).
func Gray(v uint8) Color {
	return RGB(v, v, v)
}

// GrayFloat creates a grayscale color from a float (0.0-1.0).
func GrayFloat(v float64) Color {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	b := uint8(v * 255)
	return RGB(b, b, b)
}

// Common colors
var (
	Black   = RGB(0, 0, 0)
	White   = RGB(255, 255, 255)
	Red     = RGB(255, 0, 0)
	Green   = RGB(0, 255, 0)
	Blue    = RGB(0, 0, 255)
	Yellow  = RGB(255, 255, 0)
	Cyan    = RGB(0, 255, 255)
	Magenta = RGB(255, 0, 255)
)
