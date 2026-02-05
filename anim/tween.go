// Package anim provides animation utilities for TermGL.
package anim

import (
	"time"
)

// TweenState represents the current state of a tween.
type TweenState int

const (
	TweenPending TweenState = iota
	TweenRunning
	TweenPaused
	TweenCompleted
)

// Tween represents an animation that interpolates a value over time.
type Tween struct {
	from     float64
	to       float64
	duration time.Duration
	delay    time.Duration
	easing   EasingFunc
	onUpdate func(value float64)
	onComplete func()

	state     TweenState
	startTime time.Time
	pauseTime time.Time
	elapsed   time.Duration

	// Repeat settings
	repeat      int  // -1 for infinite, 0 for no repeat, n for n times
	repeatDelay time.Duration
	yoyo        bool // Alternate direction on each repeat
	repeatCount int
	reversed    bool
}

// NewTween creates a new tween animation.
func NewTween(from, to float64, duration time.Duration) *Tween {
	return &Tween{
		from:     from,
		to:       to,
		duration: duration,
		easing:   Linear,
		state:    TweenPending,
		repeat:   0,
	}
}

// Ease sets the easing function.
func (t *Tween) Ease(easing EasingFunc) *Tween {
	t.easing = easing
	return t
}

// EaseByName sets the easing function by name.
func (t *Tween) EaseByName(name string) *Tween {
	t.easing = EasingByName(name)
	return t
}

// Delay sets the delay before the tween starts.
func (t *Tween) Delay(d time.Duration) *Tween {
	t.delay = d
	return t
}

// OnUpdate sets the callback function called on each update.
func (t *Tween) OnUpdate(fn func(value float64)) *Tween {
	t.onUpdate = fn
	return t
}

// OnComplete sets the callback function called when the tween completes.
func (t *Tween) OnComplete(fn func()) *Tween {
	t.onComplete = fn
	return t
}

// Repeat sets the number of times to repeat the tween.
// Use -1 for infinite repeats.
func (t *Tween) Repeat(count int) *Tween {
	t.repeat = count
	return t
}

// RepeatDelay sets the delay between repeats.
func (t *Tween) RepeatDelay(d time.Duration) *Tween {
	t.repeatDelay = d
	return t
}

// Yoyo enables alternating direction on each repeat.
func (t *Tween) Yoyo(enable bool) *Tween {
	t.yoyo = enable
	return t
}

// Start begins the tween animation.
func (t *Tween) Start() *Tween {
	t.startTime = time.Now()
	t.state = TweenRunning
	t.elapsed = 0
	t.repeatCount = 0
	t.reversed = false
	return t
}

// Pause pauses the tween.
func (t *Tween) Pause() *Tween {
	if t.state == TweenRunning {
		t.pauseTime = time.Now()
		t.state = TweenPaused
	}
	return t
}

// Resume resumes a paused tween.
func (t *Tween) Resume() *Tween {
	if t.state == TweenPaused {
		pauseDuration := time.Since(t.pauseTime)
		t.startTime = t.startTime.Add(pauseDuration)
		t.state = TweenRunning
	}
	return t
}

// Stop stops the tween and marks it as completed.
func (t *Tween) Stop() *Tween {
	t.state = TweenCompleted
	return t
}

// Reset resets the tween to its initial state.
func (t *Tween) Reset() *Tween {
	t.state = TweenPending
	t.elapsed = 0
	t.repeatCount = 0
	t.reversed = false
	return t
}

// Update updates the tween with the current time.
// Returns true if the tween is still running.
func (t *Tween) Update() bool {
	if t.state != TweenRunning {
		return t.state != TweenCompleted
	}

	t.elapsed = time.Since(t.startTime)

	// Handle delay
	if t.elapsed < t.delay {
		return true
	}

	effectiveElapsed := t.elapsed - t.delay

	// Calculate progress
	progress := float64(effectiveElapsed) / float64(t.duration)
	if progress > 1 {
		progress = 1
	}

	// Handle yoyo reversal
	if t.reversed {
		progress = 1 - progress
	}

	// Apply easing
	easedProgress := t.easing(progress)

	// Calculate current value
	value := Lerp(t.from, t.to, easedProgress)

	// Call update callback
	if t.onUpdate != nil {
		t.onUpdate(value)
	}

	// Check if complete
	if float64(effectiveElapsed) >= float64(t.duration) {
		// Handle repeats
		if t.repeat != 0 && (t.repeat < 0 || t.repeatCount < t.repeat) {
			t.repeatCount++
			t.startTime = time.Now().Add(-t.delay)

			if t.yoyo {
				t.reversed = !t.reversed
			}

			if t.repeatDelay > 0 {
				t.startTime = t.startTime.Add(-t.repeatDelay)
			}

			return true
		}

		t.state = TweenCompleted
		if t.onComplete != nil {
			t.onComplete()
		}
		return false
	}

	return true
}

// Value returns the current interpolated value.
func (t *Tween) Value() float64 {
	if t.state == TweenPending {
		return t.from
	}
	if t.state == TweenCompleted {
		if t.reversed {
			return t.from
		}
		return t.to
	}

	effectiveElapsed := t.elapsed - t.delay
	if effectiveElapsed < 0 {
		return t.from
	}

	progress := float64(effectiveElapsed) / float64(t.duration)
	if progress > 1 {
		progress = 1
	}

	if t.reversed {
		progress = 1 - progress
	}

	return Lerp(t.from, t.to, t.easing(progress))
}

// Progress returns the current progress (0-1).
func (t *Tween) Progress() float64 {
	if t.state == TweenPending {
		return 0
	}
	if t.state == TweenCompleted {
		return 1
	}

	effectiveElapsed := t.elapsed - t.delay
	if effectiveElapsed < 0 {
		return 0
	}

	progress := float64(effectiveElapsed) / float64(t.duration)
	if progress > 1 {
		return 1
	}
	return progress
}

// State returns the current tween state.
func (t *Tween) State() TweenState {
	return t.state
}

// IsRunning returns true if the tween is currently running.
func (t *Tween) IsRunning() bool {
	return t.state == TweenRunning
}

// IsCompleted returns true if the tween has completed.
func (t *Tween) IsCompleted() bool {
	return t.state == TweenCompleted
}

// PropertyTween animates a property by pointer.
type PropertyTween struct {
	*Tween
	target *float64
}

// NewPropertyTween creates a tween that directly updates a float64 pointer.
func NewPropertyTween(target *float64, to float64, duration time.Duration) *PropertyTween {
	from := *target
	pt := &PropertyTween{
		Tween:  NewTween(from, to, duration),
		target: target,
	}
	pt.OnUpdate(func(v float64) {
		*pt.target = v
	})
	return pt
}

// TweenGroup manages multiple tweens that run together.
type TweenGroup struct {
	tweens []*Tween
}

// NewTweenGroup creates a new tween group.
func NewTweenGroup() *TweenGroup {
	return &TweenGroup{}
}

// Add adds a tween to the group.
func (g *TweenGroup) Add(t *Tween) *TweenGroup {
	g.tweens = append(g.tweens, t)
	return g
}

// Start starts all tweens in the group.
func (g *TweenGroup) Start() *TweenGroup {
	for _, t := range g.tweens {
		t.Start()
	}
	return g
}

// Update updates all tweens in the group.
// Returns true if any tween is still running.
func (g *TweenGroup) Update() bool {
	anyRunning := false
	for _, t := range g.tweens {
		if t.Update() {
			anyRunning = true
		}
	}
	return anyRunning
}

// Stop stops all tweens in the group.
func (g *TweenGroup) Stop() *TweenGroup {
	for _, t := range g.tweens {
		t.Stop()
	}
	return g
}

// Reset resets all tweens in the group.
func (g *TweenGroup) Reset() *TweenGroup {
	for _, t := range g.tweens {
		t.Reset()
	}
	return g
}

// AllCompleted returns true if all tweens are completed.
func (g *TweenGroup) AllCompleted() bool {
	for _, t := range g.tweens {
		if !t.IsCompleted() {
			return false
		}
	}
	return true
}

// Stagger creates a group of tweens with staggered start times.
func Stagger(tweens []*Tween, staggerDelay time.Duration) *TweenGroup {
	group := NewTweenGroup()
	for i, t := range tweens {
		t.Delay(t.delay + time.Duration(i)*staggerDelay)
		group.Add(t)
	}
	return group
}

// StaggerFrom creates staggered tweens from a common starting config.
func StaggerFrom(count int, from, to float64, duration time.Duration, staggerDelay time.Duration, onUpdate func(index int, value float64)) *TweenGroup {
	group := NewTweenGroup()
	for i := 0; i < count; i++ {
		idx := i
		t := NewTween(from, to, duration).
			Delay(time.Duration(i) * staggerDelay).
			OnUpdate(func(v float64) {
				onUpdate(idx, v)
			})
		group.Add(t)
	}
	return group
}
