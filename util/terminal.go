// Package util provides terminal utilities for TermGL.
package util

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/term"
)

// GetConsoleSize returns the current terminal width and height in characters.
// Returns (columns, rows, error).
func GetConsoleSize() (int, int, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Try stderr as fallback
		width, height, err = term.GetSize(int(os.Stderr.Fd()))
		if err != nil {
			return 80, 24, err // Default fallback
		}
	}
	return width, height, nil
}

// GetConsoleSizeOrDefault returns the terminal size, using defaults if detection fails.
func GetConsoleSizeOrDefault(defaultWidth, defaultHeight int) (int, int) {
	width, height, err := GetConsoleSize()
	if err != nil {
		return defaultWidth, defaultHeight
	}
	return width, height
}

// SetWindowTitle sets the terminal window title.
// This works on most Unix terminals and Windows Terminal.
// Note: Not all terminals support this, and some may ignore it.
func SetWindowTitle(title string) error {
	// Use OSC (Operating System Command) escape sequence
	// OSC 0 ; title BEL or OSC 0 ; title ST
	_, err := fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	return err
}

// ClearScreen clears the terminal screen and moves cursor to top-left.
func ClearScreen() error {
	// ANSI escape sequence to clear screen and move cursor to home
	_, err := fmt.Fprint(os.Stdout, "\033[2J\033[H")
	return err
}

// ClearLine clears the current line.
func ClearLine() error {
	// ANSI escape sequence to clear from cursor to end of line
	_, err := fmt.Fprint(os.Stdout, "\033[2K")
	return err
}

// MoveCursor moves the cursor to the specified position (1-indexed).
func MoveCursor(row, col int) error {
	_, err := fmt.Fprintf(os.Stdout, "\033[%d;%dH", row, col)
	return err
}

// MoveCursorHome moves the cursor to the top-left corner.
func MoveCursorHome() error {
	_, err := fmt.Fprint(os.Stdout, "\033[H")
	return err
}

// HideCursor hides the terminal cursor.
func HideCursor() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?25l")
	return err
}

// ShowCursor shows the terminal cursor.
func ShowCursor() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?25h")
	return err
}

// SaveCursorPosition saves the current cursor position.
func SaveCursorPosition() error {
	_, err := fmt.Fprint(os.Stdout, "\033[s")
	return err
}

// RestoreCursorPosition restores the previously saved cursor position.
func RestoreCursorPosition() error {
	_, err := fmt.Fprint(os.Stdout, "\033[u")
	return err
}

// EnableAlternateScreen switches to the alternate screen buffer.
// This is useful for full-screen applications that want to preserve
// the original terminal content when they exit.
func EnableAlternateScreen() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?1049h")
	return err
}

// DisableAlternateScreen switches back to the main screen buffer.
func DisableAlternateScreen() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?1049l")
	return err
}

// EnableMouseTracking enables mouse tracking events.
// After calling this, the terminal will report mouse clicks and movement.
func EnableMouseTracking() error {
	// Enable button event tracking (1002) and SGR extended mode (1006)
	_, err := fmt.Fprint(os.Stdout, "\033[?1002h\033[?1006h")
	return err
}

// DisableMouseTracking disables mouse tracking events.
func DisableMouseTracking() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?1002l\033[?1006l")
	return err
}

// EnableBracketedPaste enables bracketed paste mode.
// When enabled, pasted text is wrapped in escape sequences.
func EnableBracketedPaste() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?2004h")
	return err
}

// DisableBracketedPaste disables bracketed paste mode.
func DisableBracketedPaste() error {
	_, err := fmt.Fprint(os.Stdout, "\033[?2004l")
	return err
}

// IsTerminal returns true if stdout is a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// GetPlatform returns the current operating system name.
func GetPlatform() string {
	return runtime.GOOS
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// TerminalCapabilities holds information about terminal capabilities.
type TerminalCapabilities struct {
	Width      int
	Height     int
	IsTerminal bool
	ColorDepth int // 0=no color, 1=16, 2=256, 3=truecolor
	Platform   string
}

// GetTerminalCapabilities returns information about the terminal.
func GetTerminalCapabilities() TerminalCapabilities {
	width, height, _ := GetConsoleSize()
	colorDepth := detectColorDepth()

	return TerminalCapabilities{
		Width:      width,
		Height:     height,
		IsTerminal: IsTerminal(),
		ColorDepth: colorDepth,
		Platform:   GetPlatform(),
	}
}

// detectColorDepth tries to detect the terminal's color support.
func detectColorDepth() int {
	// Check environment variables
	colorTerm := os.Getenv("COLORTERM")
	if colorTerm == "truecolor" || colorTerm == "24bit" {
		return 3 // True color (24-bit)
	}

	termEnv := os.Getenv("TERM")
	if termEnv == "" {
		return 0
	}

	// Check for 256 color support
	if contains(termEnv, "256color") || contains(termEnv, "256") {
		return 2 // 256 colors
	}

	// Most modern terminals support at least 16 colors
	if termEnv != "dumb" {
		return 1 // 16 colors
	}

	return 0
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Bell sends a bell/alert to the terminal.
func Bell() error {
	_, err := fmt.Fprint(os.Stdout, "\a")
	return err
}

// SetScrollRegion sets the scrolling region of the terminal.
// top and bottom are 1-indexed line numbers.
func SetScrollRegion(top, bottom int) error {
	_, err := fmt.Fprintf(os.Stdout, "\033[%d;%dr", top, bottom)
	return err
}

// ResetScrollRegion resets the scrolling region to the full terminal.
func ResetScrollRegion() error {
	_, err := fmt.Fprint(os.Stdout, "\033[r")
	return err
}

// ScrollUp scrolls the terminal content up by n lines.
func ScrollUp(n int) error {
	_, err := fmt.Fprintf(os.Stdout, "\033[%dS", n)
	return err
}

// ScrollDown scrolls the terminal content down by n lines.
func ScrollDown(n int) error {
	_, err := fmt.Fprintf(os.Stdout, "\033[%dT", n)
	return err
}
