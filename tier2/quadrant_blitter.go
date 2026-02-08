package tier2

// QuadrantBlitter implements 2x2 sub-cell encoding using quadrant block characters.
// Architecture doc Section 5.3
type QuadrantBlitter struct{}

// SubCellSize returns 2x2.
func (QuadrantBlitter) SubCellSize() (int, int) {
	return 2, 2
}

// quadrantTable maps 4-bit patterns to Unicode quadrant characters.
// Bit layout:
//   [0][1]  (top row)
//   [2][3]  (bottom row)
var quadrantTable = [16]rune{
	0b0000: ' ',  // Empty
	0b0001: '▘',  // Upper left
	0b0010: '▝',  // Upper right
	0b0011: '▀',  // Upper half
	0b0100: '▖',  // Lower left
	0b0101: '▌',  // Left half
	0b0110: '▞',  // Diagonal /
	0b0111: '▛',  // Missing lower right
	0b1000: '▗',  // Lower right
	0b1001: '▚',  // Diagonal \
	0b1010: '▐',  // Right half
	0b1011: '▜',  // Missing lower left
	0b1100: '▄',  // Lower half
	0b1101: '▙',  // Missing upper right
	0b1110: '▟',  // Missing upper left
	0b1111: '█',  // Full block
}

// CharForPattern returns the Unicode character for the given 4-bit pattern.
func (QuadrantBlitter) CharForPattern(pattern uint8) rune {
	return quadrantTable[pattern&0b1111]
}

// NumPatterns returns 16 (2^4).
func (QuadrantBlitter) NumPatterns() int {
	return 16
}
