package render

// OutputTier identifies the selected output mode.
// Architecture doc Section 3.4
type OutputTier int

const (
	TierSixel  OutputTier = iota // Pixel-perfect Sixel protocol
	TierKitty                    // Kitty graphics protocol (future)
	TierITerm2                   // iTerm2 inline images (future)
	TierANSI                     // ANSI subpixel encoding (universal fallback)
)

// String returns a human-readable name for the output tier.
func (t OutputTier) String() string {
	switch t {
	case TierSixel:
		return "Sixel"
	case TierKitty:
		return "Kitty"
	case TierITerm2:
		return "iTerm2"
	case TierANSI:
		return "ANSI"
	default:
		return "Unknown"
	}
}

// SelectTier chooses the best output tier based on terminal capabilities.
// Architecture doc Section 3.4
func SelectTier(caps TerminalCaps) OutputTier {
	// Prefer Sixel when available — pixel-perfect output
	if caps.Sixel {
		return TierSixel
	}

	// Kitty Graphics Protocol (for future expansion)
	if caps.KittyGraphics {
		return TierKitty
	}

	// iTerm2 Inline Image Protocol
	if caps.ITerm2 {
		return TierITerm2
	}

	// Fall back to ANSI subpixel encoding
	return TierANSI
}
