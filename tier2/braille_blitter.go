package tier2

// BrailleBlitter implements 2x4 sub-cell encoding using braille characters.
// NOTE: Braille is MONOCHROME - only foreground color is used.
// Architecture doc Section 5.3
type BrailleBlitter struct{}

// SubCellSize returns 2x4.
func (BrailleBlitter) SubCellSize() (int, int) {
	return 2, 4
}

// CharForPattern returns the Unicode braille character for the given 8-bit pattern.
// Braille patterns are in U+2800–U+28FF (256 characters).
// Standard braille bit mapping (Unicode spec):
//   [0][3]  (top row)
//   [1][4]  (row 2)
//   [2][5]  (row 3)
//   [6][7]  (bottom row)
//
// The Unicode codepoint is: U+2800 + pattern
func (BrailleBlitter) CharForPattern(pattern uint8) rune {
	// Remap from our row-major layout to braille's column-major layout
	// Our layout:
	//   [0][1]  (top row)
	//   [2][3]  (row 2)
	//   [4][5]  (row 3)
	//   [6][7]  (bottom row)
	// Braille layout (bit positions in the 8-bit value):
	//   bit0=dot1, bit1=dot2, bit2=dot3, bit3=dot4,
	//   bit4=dot5, bit5=dot6, bit6=dot7, bit7=dot8
	//   [0][3]
	//   [1][4]
	//   [2][5]
	//   [6][7]

	braillePattern := uint8(0)

	// Map our row-major pattern to braille's column-major pattern
	if pattern&(1<<0) != 0 { braillePattern |= (1 << 0) } // top-left → dot1
	if pattern&(1<<1) != 0 { braillePattern |= (1 << 3) } // top-right → dot4
	if pattern&(1<<2) != 0 { braillePattern |= (1 << 1) } // mid1-left → dot2
	if pattern&(1<<3) != 0 { braillePattern |= (1 << 4) } // mid1-right → dot5
	if pattern&(1<<4) != 0 { braillePattern |= (1 << 2) } // mid2-left → dot3
	if pattern&(1<<5) != 0 { braillePattern |= (1 << 5) } // mid2-right → dot6
	if pattern&(1<<6) != 0 { braillePattern |= (1 << 6) } // bottom-left → dot7
	if pattern&(1<<7) != 0 { braillePattern |= (1 << 7) } // bottom-right → dot8

	return rune(0x2800 + int(braillePattern))
}

// NumPatterns returns 256 (2^8).
func (BrailleBlitter) NumPatterns() int {
	return 256
}
