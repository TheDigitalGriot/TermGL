// Package canvas provides the terminal framebuffer abstraction.
package canvas

import "github.com/charmbracelet/lipgloss"

// Cell represents a single terminal cell.
type Cell struct {
	Rune       rune
	Foreground lipgloss.Color
	Background lipgloss.Color
	Bold       bool
	HasFg      bool // Whether foreground color is set
	HasBg      bool // Whether background color is set
}

// DefaultCell returns the default empty cell (space with no colors).
func DefaultCell() Cell {
	return Cell{
		Rune:  ' ',
		HasFg: false,
		HasBg: false,
		Bold:  false,
	}
}

// NewCell creates a cell with the given rune and foreground color.
func NewCell(r rune, fg lipgloss.Color) Cell {
	return Cell{
		Rune:       r,
		Foreground: fg,
		HasFg:      true,
		HasBg:      false,
		Bold:       false,
	}
}

// WithBackground returns a copy of the cell with a new background color.
func (c Cell) WithBackground(bg lipgloss.Color) Cell {
	c.Background = bg
	return c
}

// WithBold returns a copy of the cell with bold set.
func (c Cell) WithBold(bold bool) Cell {
	c.Bold = bold
	return c
}
