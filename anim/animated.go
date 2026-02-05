// Package anim provides animation utilities and Bubble Tea integration.
package anim

import (
	stdmath "math"

	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/termgl/math"
)

// AnimatedFloat wraps a float64 with spring-based animation using Harmonica.
type AnimatedFloat struct {
	current  float64
	velocity float64
	target   float64
	spring   harmonica.Spring
}

// NewAnimatedFloat creates an animated float with spring physics.
// frequency controls speed (higher = faster), damping controls oscillation (1 = critical).
// fps should match your render loop's frame rate.
func NewAnimatedFloat(initial, frequency, damping float64, fps int) *AnimatedFloat {
	return &AnimatedFloat{
		current:  initial,
		target:   initial,
		velocity: 0,
		spring:   harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

// Set sets the target value that the spring will animate towards.
func (a *AnimatedFloat) Set(target float64) {
	a.target = target
}

// Get returns the current interpolated value.
func (a *AnimatedFloat) Get() float64 {
	return a.current
}

// Target returns the target value.
func (a *AnimatedFloat) Target() float64 {
	return a.target
}

// Update advances the spring simulation by one frame.
// Call this once per frame in your update loop.
func (a *AnimatedFloat) Update() {
	a.current, a.velocity = a.spring.Update(a.current, a.velocity, a.target)
}

// IsSettled returns true if the spring has settled at the target.
func (a *AnimatedFloat) IsSettled() bool {
	const threshold = 0.001
	return stdmath.Abs(a.current-a.target) < threshold &&
		stdmath.Abs(a.velocity) < threshold
}

// Jump immediately sets the current value to the target without animation.
func (a *AnimatedFloat) Jump(value float64) {
	a.current = value
	a.target = value
	a.velocity = 0
}

// AnimatedVec3 wraps a Vec3 with spring-based animation.
type AnimatedVec3 struct {
	X, Y, Z *AnimatedFloat
}

// NewAnimatedVec3 creates an animated vector with spring physics.
func NewAnimatedVec3(initial math.Vec3, frequency, damping float64, fps int) *AnimatedVec3 {
	return &AnimatedVec3{
		X: NewAnimatedFloat(initial.X, frequency, damping, fps),
		Y: NewAnimatedFloat(initial.Y, frequency, damping, fps),
		Z: NewAnimatedFloat(initial.Z, frequency, damping, fps),
	}
}

// Set sets the target vector.
func (a *AnimatedVec3) Set(target math.Vec3) {
	a.X.Set(target.X)
	a.Y.Set(target.Y)
	a.Z.Set(target.Z)
}

// Get returns the current interpolated vector.
func (a *AnimatedVec3) Get() math.Vec3 {
	return math.Vec3{
		X: a.X.Get(),
		Y: a.Y.Get(),
		Z: a.Z.Get(),
	}
}

// Update advances the animation by one frame.
func (a *AnimatedVec3) Update() {
	a.X.Update()
	a.Y.Update()
	a.Z.Update()
}

// IsSettled returns true if all components have settled.
func (a *AnimatedVec3) IsSettled() bool {
	return a.X.IsSettled() && a.Y.IsSettled() && a.Z.IsSettled()
}

// Jump immediately sets the current value to the target without animation.
func (a *AnimatedVec3) Jump(value math.Vec3) {
	a.X.Jump(value.X)
	a.Y.Jump(value.Y)
	a.Z.Jump(value.Z)
}
