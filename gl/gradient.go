// Package gl provides 3D rendering pipeline for terminal graphics.
package gl

// Gradient maps intensity values to ASCII characters.
// Matches C TGLGradient structure from TermGL-C-Plus.
type Gradient struct {
	Chars string
}

// GradientFull is the 70-character gradient from C.
// Characters progress from darkest (space) to brightest ($).
var GradientFull = &Gradient{
	Chars: " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$",
}

// GradientMin is the 10-character minimal gradient from C.
// A simpler gradient suitable for lower-resolution displays.
var GradientMin = &Gradient{
	Chars: " .:-=+*#%@",
}

// GradientBlocks uses Unicode block characters for smooth gradients.
var GradientBlocks = &Gradient{
	Chars: " ░▒▓█",
}

// GradientShade uses Unicode shade characters.
var GradientShade = &Gradient{
	Chars: " ░▒▓",
}

// GradientDots uses dot characters of increasing size.
var GradientDots = &Gradient{
	Chars: " ·•●",
}

// Char returns the character for intensity 0-255.
// Matches C tgl_grad_char function: grad->grad[grad->length * intensity / 256U]
func (g *Gradient) Char(intensity uint8) rune {
	if len(g.Chars) == 0 {
		return ' '
	}
	idx := int(intensity) * len(g.Chars) / 256
	if idx >= len(g.Chars) {
		idx = len(g.Chars) - 1
	}
	return []rune(g.Chars)[idx]
}

// CharFloat returns the character for intensity 0.0-1.0.
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
	return len(g.Chars)
}

// NewGradient creates a custom gradient from a string of characters.
// Characters should be ordered from darkest to brightest.
func NewGradient(chars string) *Gradient {
	return &Gradient{Chars: chars}
}
