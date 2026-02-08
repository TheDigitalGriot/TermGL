package tier2

// Blitter defines a character-grid encoding strategy.
// Architecture doc Section 5.3
type Blitter interface {
	// SubCellSize returns the sub-pixel resolution within one terminal cell.
	// e.g., (2, 3) for sextant encoding means each cell is a 2×3 pixel grid.
	SubCellSize() (width, height int)

	// CharForPattern returns the Unicode character whose shape matches
	// the given binary on/off pattern.
	// pattern is a bitmask: bit i corresponds to sub-pixel i
	// (row-major, top-left = bit 0).
	CharForPattern(pattern uint8) rune

	// NumPatterns returns 2^(width*height), the number of possible patterns.
	NumPatterns() int
}

// UnicodeLevel indicates terminal Unicode support.
// Architecture doc Section 3.1
type UnicodeLevel int

const (
	UnicodeASCII UnicodeLevel = iota // ASCII only
	UnicodeBMP                        // Half-blocks, quadrants
	Unicode13                         // Sextants, braille
	Unicode16                         // Octants (2x4, 2-color)
)

// BlitterForLevel returns the best blitter for the terminal's Unicode support.
func BlitterForLevel(level UnicodeLevel) Blitter {
	switch level {
	case Unicode16:
		return SextantBlitter{} // Octant not implemented yet, sextant is next best
	case Unicode13:
		return SextantBlitter{}
	case UnicodeBMP:
		return QuadrantBlitter{}
	default:
		return HalfBlockBlitter{}
	}
}
