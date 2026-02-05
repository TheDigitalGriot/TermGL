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

// Common colors (standard ANSI)
var (
	Black   = lipgloss.Color("0")
	Red     = lipgloss.Color("1")
	Green   = lipgloss.Color("2")
	Yellow  = lipgloss.Color("3")
	Blue    = lipgloss.Color("4")
	Magenta = lipgloss.Color("5")
	Cyan    = lipgloss.Color("6")
	White   = lipgloss.Color("7")
)

// Bright/High-intensity colors (ANSI 8-15)
var (
	BrightBlack   = lipgloss.Color("8")
	BrightRed     = lipgloss.Color("9")
	BrightGreen   = lipgloss.Color("10")
	BrightYellow  = lipgloss.Color("11")
	BrightBlue    = lipgloss.Color("12")
	BrightMagenta = lipgloss.Color("13")
	BrightCyan    = lipgloss.Color("14")
	BrightWhite   = lipgloss.Color("15")
)
