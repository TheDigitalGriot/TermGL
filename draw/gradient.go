// Package draw provides 2D drawing primitives for the canvas.
package draw

// Gradient represents a sequence of characters ordered from dark to bright.
// Used to map intensity values (0-255) to visual characters.
type Gradient struct {
	chars []rune
}

// GradientFull is the full 70-character gradient from dark to bright.
// From wojciech-graj/TermGL: ` .'`^",:;Il!i><~+_-?][}{1)(|\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$`
var GradientFull = NewGradient(" .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$")

// GradientMin is the minimal 10-character gradient.
// This is the default used by the 3D renderer.
var GradientMin = NewGradient(" .:-=+*#%@")

// GradientBlocks uses Unicode block characters for smooth gradients.
var GradientBlocks = NewGradient(" ░▒▓█")

// GradientDots uses dots/periods for a subtle gradient.
var GradientDots = NewGradient(" ·•●")

// GradientAscii uses only basic ASCII for maximum compatibility.
var GradientAscii = NewGradient(" .-:=+*#%@")

// GradientShade uses Unicode shade characters.
var GradientShade = NewGradient(" ░▒▓")

// NewGradient creates a new gradient from a string of characters.
// Characters should be ordered from dark (low intensity) to bright (high intensity).
func NewGradient(chars string) *Gradient {
	return &Gradient{
		chars: []rune(chars),
	}
}

// Char returns the character corresponding to an intensity value (0-255).
// 0 = darkest, 255 = brightest.
func (g *Gradient) Char(intensity uint8) rune {
	if len(g.chars) == 0 {
		return ' '
	}
	index := int(intensity) * (len(g.chars) - 1) / 255
	if index >= len(g.chars) {
		index = len(g.chars) - 1
	}
	return g.chars[index]
}

// CharFloat returns the character corresponding to an intensity value (0.0-1.0).
func (g *Gradient) CharFloat(intensity float64) rune {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}
	return g.Char(uint8(intensity * 255))
}

// Length returns the number of characters in the gradient.
func (g *Gradient) Length() int {
	return len(g.chars)
}

// Chars returns the gradient characters as a string.
func (g *Gradient) Chars() string {
	return string(g.chars)
}

// Reversed returns a new gradient with characters in reverse order.
// This is useful when you want bright characters to represent low intensity.
func (g *Gradient) Reversed() *Gradient {
	reversed := make([]rune, len(g.chars))
	for i, r := range g.chars {
		reversed[len(g.chars)-1-i] = r
	}
	return &Gradient{chars: reversed}
}

// Subset returns a new gradient using only a portion of the original.
// start and end are indices (0 to Length()-1).
func (g *Gradient) Subset(start, end int) *Gradient {
	if start < 0 {
		start = 0
	}
	if end > len(g.chars) {
		end = len(g.chars)
	}
	if start >= end {
		return NewGradient(" ")
	}
	return &Gradient{chars: g.chars[start:end]}
}

// IntensityToChar is a convenience function to get a gradient character.
// Uses the full gradient by default.
func IntensityToChar(intensity uint8) rune {
	return GradientFull.Char(intensity)
}

// IntensityToCharFloat is a convenience function using float intensity (0.0-1.0).
func IntensityToCharFloat(intensity float64) rune {
	return GradientFull.CharFloat(intensity)
}
