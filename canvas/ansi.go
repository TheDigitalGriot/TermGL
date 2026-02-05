package canvas

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// String renders the canvas to an ANSI string for Bubble Tea's View().
// This uses lipgloss for styling to ensure compatibility with the Charm ecosystem.
func (c *Canvas) String() string {
	var sb strings.Builder
	sb.Grow(c.width * c.height * 20) // Estimate capacity

	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			cell := c.GetCell(x, y)

			// Build style for this cell
			style := lipgloss.NewStyle()

			// Apply foreground color if set
			if cell.HasFg {
				style = style.Foreground(cell.Foreground)
			}

			// Apply background color if set
			if cell.HasBg {
				style = style.Background(cell.Background)
			}

			// Apply bold if set
			if cell.Bold {
				style = style.Bold(true)
			}

			sb.WriteString(style.Render(string(cell.Rune)))
		}
		if y < c.height-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// StringSimple renders the canvas without colors (for debugging).
func (c *Canvas) StringSimple() string {
	var sb strings.Builder
	sb.Grow(c.width*c.height + c.height)

	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			sb.WriteRune(c.GetCell(x, y).Rune)
		}
		if y < c.height-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
