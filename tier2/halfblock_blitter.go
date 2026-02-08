package tier2

// HalfBlockBlitter implements 1x2 sub-cell encoding using half-block characters.
// Architecture doc Section 5.3
type HalfBlockBlitter struct{}

// SubCellSize returns 1x2 (one pixel wide, two pixels tall).
func (HalfBlockBlitter) SubCellSize() (int, int) {
	return 1, 2
}

// CharForPattern returns the Unicode character for the given 2-bit pattern.
// Pattern bits:
//   bit 0 = top pixel
//   bit 1 = bottom pixel
func (HalfBlockBlitter) CharForPattern(pattern uint8) rune {
	switch pattern & 0b11 {
	case 0b00: // Both off
		return ' '
	case 0b01: // Top off, bottom on
		return '▄'
	case 0b10: // Top on, bottom off
		return '▀'
	case 0b11: // Both on
		return '█'
	default:
		return ' '
	}
}

// NumPatterns returns 4 (2^2).
func (HalfBlockBlitter) NumPatterns() int {
	return 4
}
