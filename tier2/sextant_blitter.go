package tier2

// SextantBlitter implements 2x3 sub-cell encoding using sextant block characters.
// Unicode 13.0 Symbols for Legacy Computing (U+1FB00–U+1FB3B).
// Architecture doc Section 5.3
type SextantBlitter struct{}

// SubCellSize returns 2x3.
func (SextantBlitter) SubCellSize() (int, int) {
	return 2, 3
}

// sextantTable maps 6-bit patterns to Unicode sextant characters.
// Bit layout within a cell:
//   [0][1]  (top row)
//   [2][3]  (middle row)
//   [4][5]  (bottom row)
var sextantTable = [64]rune{
	0b000000: ' ',
	0b000001: '\U0001FB00', // 🬀
	0b000010: '\U0001FB01', // 🬁
	0b000011: '\U0001FB02', // 🬂
	0b000100: '\U0001FB03', // 🬃
	0b000101: '\U0001FB04', // 🬄
	0b000110: '\U0001FB05', // 🬅
	0b000111: '\U0001FB06', // 🬆
	0b001000: '\U0001FB07', // 🬇
	0b001001: '\U0001FB08', // 🬈
	0b001010: '\U0001FB09', // 🬉
	0b001011: '\U0001FB0A', // 🬊
	0b001100: '\U0001FB0B', // 🬋
	0b001101: '\U0001FB0C', // 🬌
	0b001110: '\U0001FB0D', // 🬍
	0b001111: '\U0001FB0E', // 🬎
	0b010000: '\U0001FB0F', // 🬏
	0b010001: '\U0001FB10', // 🬐
	0b010010: '\U0001FB11', // 🬑
	0b010011: '\U0001FB12', // 🬒
	0b010100: '\U0001FB13', // 🬓
	0b010101: '▌',          // Left half (standard block element)
	0b010110: '\U0001FB14', // 🬔
	0b010111: '\U0001FB15', // 🬕
	0b011000: '\U0001FB16', // 🬖
	0b011001: '\U0001FB17', // 🬗
	0b011010: '▐',          // Right half (standard block element)
	0b011011: '\U0001FB18', // 🬘
	0b011100: '\U0001FB19', // 🬙
	0b011101: '\U0001FB1A', // 🬚
	0b011110: '\U0001FB1B', // 🬛
	0b011111: '\U0001FB1C', // 🬜
	0b100000: '\U0001FB1D', // 🬝
	0b100001: '\U0001FB1E', // 🬞
	0b100010: '\U0001FB1F', // 🬟
	0b100011: '\U0001FB20', // 🬠
	0b100100: '\U0001FB21', // 🬡
	0b100101: '\U0001FB22', // 🬢
	0b100110: '\U0001FB23', // 🬣
	0b100111: '\U0001FB24', // 🬤
	0b101000: '\U0001FB25', // 🬥
	0b101001: '\U0001FB26', // 🬦
	0b101010: '\U0001FB27', // 🬧
	0b101011: '▜',          // Upper right 3/4 (standard block element)
	0b101100: '\U0001FB28', // 🬨
	0b101101: '▛',          // Upper left 3/4 (standard block element)
	0b101110: '▟',          // Lower left 3/4 (standard block element)
	0b101111: '\U0001FB29', // 🬩
	0b110000: '\U0001FB2A', // 🬪
	0b110001: '\U0001FB2B', // 🬫
	0b110010: '\U0001FB2C', // 🬬
	0b110011: '▀',          // Upper half (standard block element)
	0b110100: '\U0001FB2D', // 🬭
	0b110101: '\U0001FB2E', // 🬮
	0b110110: '\U0001FB2F', // 🬯
	0b110111: '\U0001FB30', // 🬰
	0b111000: '\U0001FB31', // 🬱
	0b111001: '\U0001FB32', // 🬲
	0b111010: '\U0001FB33', // 🬳
	0b111011: '\U0001FB34', // 🬴
	0b111100: '▄',          // Lower half (standard block element)
	0b111101: '\U0001FB35', // 🬵
	0b111110: '\U0001FB36', // 🬶
	0b111111: '█',          // Full block (standard block element)
}

// CharForPattern returns the Unicode character for the given 6-bit pattern.
func (SextantBlitter) CharForPattern(pattern uint8) rune {
	return sextantTable[pattern&0b111111]
}

// NumPatterns returns 64 (2^6).
func (SextantBlitter) NumPatterns() int {
	return 64
}
