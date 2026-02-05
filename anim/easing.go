// Package anim provides animation utilities for TermGL.
package anim

import (
	"math"
)

// EasingFunc is a function that transforms a linear progress value (0-1)
// into an eased value. The input and output are both in the range [0, 1].
type EasingFunc func(t float64) float64

// Standard easing functions
// These follow the CSS/GSAP naming conventions.

// Linear returns the input unchanged.
func Linear(t float64) float64 {
	return t
}

// --- Quadratic (power of 2) ---

// EaseInQuad accelerates from zero velocity.
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutQuad decelerates to zero velocity.
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutQuad accelerates until halfway, then decelerates.
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// --- Cubic (power of 3) ---

// EaseInCubic accelerates from zero velocity.
func EaseInCubic(t float64) float64 {
	return t * t * t
}

// EaseOutCubic decelerates to zero velocity.
func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

// EaseInOutCubic accelerates until halfway, then decelerates.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t = t*2 - 2
	return (t*t*t + 2) / 2
}

// --- Quartic (power of 4) ---

// EaseInQuart accelerates from zero velocity.
func EaseInQuart(t float64) float64 {
	return t * t * t * t
}

// EaseOutQuart decelerates to zero velocity.
func EaseOutQuart(t float64) float64 {
	t--
	return 1 - t*t*t*t
}

// EaseInOutQuart accelerates until halfway, then decelerates.
func EaseInOutQuart(t float64) float64 {
	if t < 0.5 {
		return 8 * t * t * t * t
	}
	t = t*2 - 2
	return 1 - t*t*t*t/2
}

// --- Quintic (power of 5) ---

// EaseInQuint accelerates from zero velocity.
func EaseInQuint(t float64) float64 {
	return t * t * t * t * t
}

// EaseOutQuint decelerates to zero velocity.
func EaseOutQuint(t float64) float64 {
	t--
	return t*t*t*t*t + 1
}

// EaseInOutQuint accelerates until halfway, then decelerates.
func EaseInOutQuint(t float64) float64 {
	if t < 0.5 {
		return 16 * t * t * t * t * t
	}
	t = t*2 - 2
	return (t*t*t*t*t + 2) / 2
}

// --- Sinusoidal ---

// EaseInSine accelerates using a sine function.
func EaseInSine(t float64) float64 {
	return 1 - math.Cos(t*math.Pi/2)
}

// EaseOutSine decelerates using a sine function.
func EaseOutSine(t float64) float64 {
	return math.Sin(t * math.Pi / 2)
}

// EaseInOutSine accelerates and decelerates using sine.
func EaseInOutSine(t float64) float64 {
	return (1 - math.Cos(math.Pi*t)) / 2
}

// --- Exponential ---

// EaseInExpo accelerates exponentially.
func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

// EaseOutExpo decelerates exponentially.
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// EaseInOutExpo accelerates and decelerates exponentially.
func EaseInOutExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

// --- Circular ---

// EaseInCirc accelerates using a circular curve.
func EaseInCirc(t float64) float64 {
	return 1 - math.Sqrt(1-t*t)
}

// EaseOutCirc decelerates using a circular curve.
func EaseOutCirc(t float64) float64 {
	t--
	return math.Sqrt(1 - t*t)
}

// EaseInOutCirc accelerates and decelerates using circular curves.
func EaseInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-4*t*t)) / 2
	}
	t = t*2 - 2
	return (math.Sqrt(1-t*t) + 1) / 2
}

// --- Elastic ---

// EaseInElastic has an elastic effect at the start.
func EaseInElastic(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	return -math.Pow(2, 10*(t-1)) * math.Sin((t-1.1)*5*math.Pi)
}

// EaseOutElastic has an elastic effect at the end.
func EaseOutElastic(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	return math.Pow(2, -10*t)*math.Sin((t-0.1)*5*math.Pi) + 1
}

// EaseInOutElastic has elastic effects at both ends.
func EaseInOutElastic(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	t *= 2
	if t < 1 {
		return -0.5 * math.Pow(2, 10*(t-1)) * math.Sin((t-1.1)*5*math.Pi)
	}
	return 0.5*math.Pow(2, -10*(t-1))*math.Sin((t-1.1)*5*math.Pi) + 1
}

// --- Back (overshooting) ---

const backOvershoot = 1.70158

// EaseInBack overshoots at the start.
func EaseInBack(t float64) float64 {
	return t * t * ((backOvershoot+1)*t - backOvershoot)
}

// EaseOutBack overshoots at the end.
func EaseOutBack(t float64) float64 {
	t--
	return t*t*((backOvershoot+1)*t+backOvershoot) + 1
}

// EaseInOutBack overshoots at both ends.
func EaseInOutBack(t float64) float64 {
	s := backOvershoot * 1.525
	t *= 2
	if t < 1 {
		return (t * t * ((s+1)*t - s)) / 2
	}
	t -= 2
	return (t*t*((s+1)*t+s) + 2) / 2
}

// --- Bounce ---

// EaseInBounce has a bouncing effect at the start.
func EaseInBounce(t float64) float64 {
	return 1 - EaseOutBounce(1-t)
}

// EaseOutBounce has a bouncing effect at the end.
func EaseOutBounce(t float64) float64 {
	if t < 1/2.75 {
		return 7.5625 * t * t
	}
	if t < 2/2.75 {
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	}
	if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	}
	t -= 2.625 / 2.75
	return 7.5625*t*t + 0.984375
}

// EaseInOutBounce has bouncing effects at both ends.
func EaseInOutBounce(t float64) float64 {
	if t < 0.5 {
		return EaseInBounce(t*2) / 2
	}
	return EaseOutBounce(t*2-1)/2 + 0.5
}

// --- Steps ---

// Steps creates a stepped easing function that jumps in discrete steps.
func Steps(numSteps int, jumpStart bool) EasingFunc {
	return func(t float64) float64 {
		step := math.Floor(t * float64(numSteps))
		if jumpStart {
			step++
		}
		return step / float64(numSteps)
	}
}

// StepsEnd creates a stepped easing that jumps at the end of each interval.
func StepsEnd(numSteps int) EasingFunc {
	return Steps(numSteps, false)
}

// StepsStart creates a stepped easing that jumps at the start of each interval.
func StepsStart(numSteps int) EasingFunc {
	return Steps(numSteps, true)
}

// --- Cubic Bezier ---

// CubicBezier creates a custom cubic bezier easing function.
// Control points are (0,0), (x1,y1), (x2,y2), (1,1).
// This matches CSS cubic-bezier().
func CubicBezier(x1, y1, x2, y2 float64) EasingFunc {
	return func(t float64) float64 {
		// Newton-Raphson iteration to find t for given x
		const epsilon = 1e-6
		const maxIterations = 8

		// Initial guess
		x := t

		for i := 0; i < maxIterations; i++ {
			// Calculate bezier x value at current t
			bx := bezierX(x, x1, x2)
			dx := bx - t

			if math.Abs(dx) < epsilon {
				break
			}

			// Calculate derivative
			dbx := bezierDx(x, x1, x2)
			if math.Abs(dbx) < epsilon {
				break
			}

			x -= dx / dbx
		}

		return bezierY(x, y1, y2)
	}
}

func bezierX(t, x1, x2 float64) float64 {
	// B(t) = 3(1-t)^2*t*x1 + 3(1-t)*t^2*x2 + t^3
	return 3*(1-t)*(1-t)*t*x1 + 3*(1-t)*t*t*x2 + t*t*t
}

func bezierY(t, y1, y2 float64) float64 {
	return 3*(1-t)*(1-t)*t*y1 + 3*(1-t)*t*t*y2 + t*t*t
}

func bezierDx(t, x1, x2 float64) float64 {
	// Derivative of bezier x with respect to t
	return 3*(1-t)*(1-t)*x1 + 6*(1-t)*t*(x2-x1) + 3*t*t*(1-x2)
}

// --- Named Easing Presets (CSS equivalents) ---

// Ease is the default CSS easing (equivalent to cubic-bezier(0.25, 0.1, 0.25, 1.0))
var Ease = CubicBezier(0.25, 0.1, 0.25, 1.0)

// EaseIn is the CSS ease-in (equivalent to cubic-bezier(0.42, 0, 1.0, 1.0))
var EaseIn = CubicBezier(0.42, 0, 1.0, 1.0)

// EaseOut is the CSS ease-out (equivalent to cubic-bezier(0, 0, 0.58, 1.0))
var EaseOut = CubicBezier(0, 0, 0.58, 1.0)

// EaseInOut is the CSS ease-in-out (equivalent to cubic-bezier(0.42, 0, 0.58, 1.0))
var EaseInOut = CubicBezier(0.42, 0, 0.58, 1.0)

// --- Easing lookup by name ---

// EasingByName returns an easing function by its name.
// Returns Linear if the name is not recognized.
func EasingByName(name string) EasingFunc {
	switch name {
	case "linear":
		return Linear
	case "easeInQuad":
		return EaseInQuad
	case "easeOutQuad":
		return EaseOutQuad
	case "easeInOutQuad":
		return EaseInOutQuad
	case "easeInCubic":
		return EaseInCubic
	case "easeOutCubic":
		return EaseOutCubic
	case "easeInOutCubic":
		return EaseInOutCubic
	case "easeInQuart":
		return EaseInQuart
	case "easeOutQuart":
		return EaseOutQuart
	case "easeInOutQuart":
		return EaseInOutQuart
	case "easeInQuint":
		return EaseInQuint
	case "easeOutQuint":
		return EaseOutQuint
	case "easeInOutQuint":
		return EaseInOutQuint
	case "easeInSine":
		return EaseInSine
	case "easeOutSine":
		return EaseOutSine
	case "easeInOutSine":
		return EaseInOutSine
	case "easeInExpo":
		return EaseInExpo
	case "easeOutExpo":
		return EaseOutExpo
	case "easeInOutExpo":
		return EaseInOutExpo
	case "easeInCirc":
		return EaseInCirc
	case "easeOutCirc":
		return EaseOutCirc
	case "easeInOutCirc":
		return EaseInOutCirc
	case "easeInElastic":
		return EaseInElastic
	case "easeOutElastic":
		return EaseOutElastic
	case "easeInOutElastic":
		return EaseInOutElastic
	case "easeInBack":
		return EaseInBack
	case "easeOutBack":
		return EaseOutBack
	case "easeInOutBack":
		return EaseInOutBack
	case "easeInBounce":
		return EaseInBounce
	case "easeOutBounce":
		return EaseOutBounce
	case "easeInOutBounce":
		return EaseInOutBounce
	case "ease":
		return Ease
	case "easeIn":
		return EaseIn
	case "easeOut":
		return EaseOut
	case "easeInOut":
		return EaseInOut
	default:
		return Linear
	}
}

// Clamp ensures t is within [0, 1].
func Clamp(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// Lerp performs linear interpolation between a and b based on t.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// InverseLerp returns the t value that would produce v when lerping from a to b.
func InverseLerp(a, b, v float64) float64 {
	if a == b {
		return 0
	}
	return (v - a) / (b - a)
}

// Remap maps a value from one range to another.
func Remap(v, inMin, inMax, outMin, outMax float64) float64 {
	t := InverseLerp(inMin, inMax, v)
	return Lerp(outMin, outMax, t)
}
