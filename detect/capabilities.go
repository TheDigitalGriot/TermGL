// Package detect provides terminal capability detection for adaptive rendering.
// It determines what encoding levels the current terminal supports.
package detect

import (
	"image"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// EncoderLevel represents the pixel encoding strategy
type EncoderLevel int

const (
	LevelCharacter EncoderLevel = iota // Level 0: character luminance ramp
	LevelHalfBlock                     // Level 1: ▀▄ with 24-bit color
	LevelQuadrant                      // Level 2: ▖▗▘▙ quadrant blocks
	LevelBraille                       // Level 3: ⠀-⣿ braille subpixels
	LevelSixel                         // Level 4: sixel bitmaps
	LevelKitty                         // Level 5: kitty graphics protocol
	LevelITerm                         // Level 5b: iTerm2 inline images
)

// String returns a human-readable name for the encoder level
func (l EncoderLevel) String() string {
	switch l {
	case LevelCharacter:
		return "Character"
	case LevelHalfBlock:
		return "HalfBlock"
	case LevelQuadrant:
		return "Quadrant"
	case LevelBraille:
		return "Braille"
	case LevelSixel:
		return "Sixel"
	case LevelKitty:
		return "Kitty"
	case LevelITerm:
		return "iTerm"
	default:
		return "Unknown"
	}
}

// TerminalCaps holds detected terminal capabilities
type TerminalCaps struct {
	// ColorDepth: 2, 8, 16, 256, or 16777216 (24-bit)
	ColorDepth int

	// HasUnicode indicates support for Unicode block elements (▀▄)
	HasUnicode bool

	// HasBraille indicates support for braille characters (⠀-⣿)
	HasBraille bool

	// HasSixel indicates sixel graphics support
	HasSixel bool

	// HasKitty indicates kitty graphics protocol support
	HasKitty bool

	// HasITerm indicates iTerm2 inline images support
	HasITerm bool

	// HasSyncOutput indicates DEC synchronized output mode support
	HasSyncOutput bool

	// CellSize is the pixel dimensions of one character cell
	CellSize image.Point

	// GridSize is the terminal dimensions in cells (cols × rows)
	GridSize image.Point

	// BestLevel is the highest supported encoding level
	BestLevel EncoderLevel

	// OS is "windows", "darwin", or "linux"
	OS string

	// Terminal is the detected terminal name
	Terminal string
}

// Capabilities detects and returns the current terminal's capabilities.
// Results are computed fresh each call (not cached).
func Capabilities() *TerminalCaps {
	caps := &TerminalCaps{
		OS: runtime.GOOS,
	}

	// Detect terminal type from environment
	caps.detectTerminal()

	// Get terminal grid size
	caps.detectGridSize()

	// Determine capabilities based on terminal
	caps.determineCapabilities()

	// Calculate best encoding level
	caps.BestLevel = caps.determineBestLevel()

	return caps
}

// detectTerminal identifies the terminal from environment variables
func (c *TerminalCaps) detectTerminal() {
	// Check common environment variables
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")
	wtSession := os.Getenv("WT_SESSION")
	kittyWindow := os.Getenv("KITTY_WINDOW_ID")
	colorTerm := os.Getenv("COLORTERM")

	switch {
	case kittyWindow != "":
		c.Terminal = "kitty"
	case termProgram == "iTerm.app":
		c.Terminal = "iterm2"
	case termProgram == "WezTerm":
		c.Terminal = "wezterm"
	case termProgram == "vscode":
		c.Terminal = "vscode"
	case termProgram == "Ghostty":
		c.Terminal = "ghostty"
	case wtSession != "":
		c.Terminal = "windows-terminal"
	case c.OS == "windows":
		c.Terminal = "conhost"
	case strings.Contains(term, "xterm"):
		c.Terminal = "xterm"
	case colorTerm == "truecolor" || colorTerm == "24bit":
		c.Terminal = "truecolor-unknown"
	default:
		c.Terminal = "unknown"
	}
}

// detectGridSize gets terminal dimensions
func (c *TerminalCaps) detectGridSize() {
	// Try to get terminal size from stdin/stdout
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to stdin
		width, height, err = term.GetSize(int(os.Stdin.Fd()))
	}
	if err != nil {
		// Default fallback
		width, height = 80, 24
	}
	c.GridSize = image.Point{X: width, Y: height}

	// Estimate cell size (most terminals use roughly 8x16 pixels per cell)
	// This is a reasonable default; exact size requires terminal queries
	c.CellSize = image.Point{X: 8, Y: 16}
}

// determineCapabilities sets capability flags based on terminal type
func (c *TerminalCaps) determineCapabilities() {
	// Default to basic capabilities
	c.ColorDepth = 16 // 16 ANSI colors
	c.HasUnicode = false
	c.HasBraille = false
	c.HasSixel = false
	c.HasKitty = false
	c.HasITerm = false
	c.HasSyncOutput = false

	switch c.Terminal {
	case "kitty":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasKitty = true
		c.HasSyncOutput = true

	case "iterm2":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasITerm = true

	case "wezterm":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasSixel = true
		c.HasITerm = true
		c.HasSyncOutput = true

	case "windows-terminal":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasSixel = true // WT 1.22+
		c.HasSyncOutput = true

	case "conhost":
		// Windows console host (legacy PowerShell)
		c.ColorDepth = 16777216 // Win10 1809+
		c.HasUnicode = true
		c.HasBraille = true
		// No sixel, no kitty, no sync output

	case "vscode":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasITerm = true // VSCode supports inline images

	case "ghostty":
		c.ColorDepth = 16777216
		c.HasUnicode = true
		c.HasBraille = true
		c.HasSyncOutput = true
		// Ghostty has limited Kitty support (static only)

	case "xterm", "truecolor-unknown":
		// Check COLORTERM for truecolor hint
		colorTerm := os.Getenv("COLORTERM")
		if colorTerm == "truecolor" || colorTerm == "24bit" {
			c.ColorDepth = 16777216
		} else {
			c.ColorDepth = 256
		}
		c.HasUnicode = true
		c.HasBraille = true

	default:
		// Unknown terminal - assume 256 colors and Unicode
		c.ColorDepth = 256
		c.HasUnicode = true
		c.HasBraille = true
	}

	// Override with environment hints
	if os.Getenv("COLORTERM") == "truecolor" || os.Getenv("COLORTERM") == "24bit" {
		c.ColorDepth = 16777216
	}
}

// determineBestLevel calculates the highest usable encoding level
func (c *TerminalCaps) determineBestLevel() EncoderLevel {
	// Check from highest to lowest
	if c.HasKitty {
		return LevelKitty
	}
	if c.HasITerm {
		return LevelITerm
	}
	if c.HasSixel {
		return LevelSixel
	}
	if c.HasBraille && c.ColorDepth >= 16777216 {
		return LevelBraille
	}
	if c.HasUnicode && c.ColorDepth >= 16777216 {
		return LevelHalfBlock
	}
	if c.HasUnicode && c.ColorDepth >= 256 {
		return LevelHalfBlock
	}
	return LevelCharacter
}

// VirtualResolution calculates the virtual pixel resolution for a given level
func (c *TerminalCaps) VirtualResolution(level EncoderLevel) image.Point {
	cols := c.GridSize.X
	rows := c.GridSize.Y

	switch level {
	case LevelCharacter:
		// 1 pixel per cell
		return image.Point{X: cols, Y: rows}
	case LevelHalfBlock:
		// 1×2 pixels per cell (vertical doubling)
		return image.Point{X: cols, Y: rows * 2}
	case LevelQuadrant:
		// 2×2 pixels per cell
		return image.Point{X: cols * 2, Y: rows * 2}
	case LevelBraille:
		// 2×4 pixels per cell
		return image.Point{X: cols * 2, Y: rows * 4}
	case LevelSixel, LevelKitty, LevelITerm:
		// True pixel resolution
		return image.Point{
			X: cols * c.CellSize.X,
			Y: rows * c.CellSize.Y,
		}
	default:
		return image.Point{X: cols, Y: rows}
	}
}

// Is24Bit returns true if the terminal supports 24-bit color
func (c *TerminalCaps) Is24Bit() bool {
	return c.ColorDepth >= 16777216
}
