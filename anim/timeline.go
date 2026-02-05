// Package anim provides animation utilities for TermGL.
package anim

import (
	"time"
)

// TimelineState represents the state of a timeline.
type TimelineState int

const (
	TimelinePending TimelineState = iota
	TimelineRunning
	TimelinePaused
	TimelineCompleted
)

// TimelineEntry represents a scheduled animation on the timeline.
type TimelineEntry struct {
	tween    *Tween
	position time.Duration // When this entry starts on the timeline
	label    string        // Optional label for seeking
}

// Timeline manages a sequence of animations with precise timing control.
// Inspired by GSAP's Timeline API.
type Timeline struct {
	entries    []*TimelineEntry
	labels     map[string]time.Duration
	duration   time.Duration
	state      TimelineState
	startTime  time.Time
	pauseTime  time.Time
	elapsed    time.Duration
	playhead   time.Duration
	timeScale  float64
	onUpdate   func(progress float64)
	onComplete func()

	// Repeat settings
	repeat      int
	repeatDelay time.Duration
	yoyo        bool
	repeatCount int
	reversed    bool
}

// NewTimeline creates a new timeline.
func NewTimeline() *Timeline {
	return &Timeline{
		entries:   make([]*TimelineEntry, 0),
		labels:    make(map[string]time.Duration),
		timeScale: 1.0,
		state:     TimelinePending,
	}
}

// Add adds a tween to the timeline at the current position (end of timeline).
func (tl *Timeline) Add(t *Tween) *Timeline {
	return tl.AddAt(t, tl.duration)
}

// AddAt adds a tween at a specific position on the timeline.
func (tl *Timeline) AddAt(t *Tween, position time.Duration) *Timeline {
	entry := &TimelineEntry{
		tween:    t,
		position: position,
	}
	tl.entries = append(tl.entries, entry)

	// Update total duration
	entryEnd := position + t.duration + t.delay
	if entryEnd > tl.duration {
		tl.duration = entryEnd
	}

	return tl
}

// AddRelative adds a tween relative to the current end position.
// offset can be negative to overlap with previous animations.
func (tl *Timeline) AddRelative(t *Tween, offset time.Duration) *Timeline {
	position := tl.duration + offset
	if position < 0 {
		position = 0
	}
	return tl.AddAt(t, position)
}

// AddLabel adds a named position on the timeline.
func (tl *Timeline) AddLabel(name string, position time.Duration) *Timeline {
	tl.labels[name] = position
	return tl
}

// AddLabelAtEnd adds a label at the current end of the timeline.
func (tl *Timeline) AddLabelAtEnd(name string) *Timeline {
	return tl.AddLabel(name, tl.duration)
}

// AddAtLabel adds a tween at a labeled position.
func (tl *Timeline) AddAtLabel(t *Tween, label string) *Timeline {
	if pos, ok := tl.labels[label]; ok {
		return tl.AddAt(t, pos)
	}
	// If label doesn't exist, add at end
	return tl.Add(t)
}

// To creates and adds a tween from the current value to the target.
// This is a convenience method similar to GSAP's .to()
func (tl *Timeline) To(target *float64, to float64, duration time.Duration, opts ...TweenOption) *Timeline {
	from := *target
	t := NewTween(from, to, duration)
	t.OnUpdate(func(v float64) {
		*target = v
	})

	for _, opt := range opts {
		opt(t)
	}

	return tl.Add(t)
}

// From creates and adds a tween from a start value to the current value.
func (tl *Timeline) From(target *float64, from float64, duration time.Duration, opts ...TweenOption) *Timeline {
	to := *target
	t := NewTween(from, to, duration)
	t.OnUpdate(func(v float64) {
		*target = v
	})

	for _, opt := range opts {
		opt(t)
	}

	return tl.Add(t)
}

// FromTo creates and adds a tween from one value to another.
func (tl *Timeline) FromTo(target *float64, from, to float64, duration time.Duration, opts ...TweenOption) *Timeline {
	t := NewTween(from, to, duration)
	t.OnUpdate(func(v float64) {
		*target = v
	})

	for _, opt := range opts {
		opt(t)
	}

	return tl.Add(t)
}

// TweenOption is a function that configures a tween.
type TweenOption func(*Tween)

// WithEase returns a TweenOption that sets the easing function.
func WithEase(easing EasingFunc) TweenOption {
	return func(t *Tween) {
		t.easing = easing
	}
}

// WithDelay returns a TweenOption that sets the delay.
func WithDelay(d time.Duration) TweenOption {
	return func(t *Tween) {
		t.delay = d
	}
}

// WithOnComplete returns a TweenOption that sets the completion callback.
func WithOnComplete(fn func()) TweenOption {
	return func(t *Tween) {
		t.onComplete = fn
	}
}

// TimeScale sets the playback speed multiplier.
func (tl *Timeline) TimeScale(scale float64) *Timeline {
	tl.timeScale = scale
	return tl
}

// OnUpdate sets the callback called on each frame with the progress (0-1).
func (tl *Timeline) OnUpdate(fn func(progress float64)) *Timeline {
	tl.onUpdate = fn
	return tl
}

// OnComplete sets the callback called when the timeline completes.
func (tl *Timeline) OnComplete(fn func()) *Timeline {
	tl.onComplete = fn
	return tl
}

// Repeat sets the number of times to repeat (-1 for infinite).
func (tl *Timeline) Repeat(count int) *Timeline {
	tl.repeat = count
	return tl
}

// RepeatDelay sets the delay between repeats.
func (tl *Timeline) RepeatDelay(d time.Duration) *Timeline {
	tl.repeatDelay = d
	return tl
}

// Yoyo enables alternating direction on repeats.
func (tl *Timeline) Yoyo(enable bool) *Timeline {
	tl.yoyo = enable
	return tl
}

// Duration returns the total duration of the timeline.
func (tl *Timeline) Duration() time.Duration {
	return tl.duration
}

// Start begins playback of the timeline.
func (tl *Timeline) Start() *Timeline {
	tl.startTime = time.Now()
	tl.state = TimelineRunning
	tl.elapsed = 0
	tl.playhead = 0
	tl.repeatCount = 0
	tl.reversed = false

	// Reset all tweens
	for _, entry := range tl.entries {
		entry.tween.Reset()
	}

	return tl
}

// Pause pauses the timeline.
func (tl *Timeline) Pause() *Timeline {
	if tl.state == TimelineRunning {
		tl.pauseTime = time.Now()
		tl.state = TimelinePaused
	}
	return tl
}

// Resume resumes a paused timeline.
func (tl *Timeline) Resume() *Timeline {
	if tl.state == TimelinePaused {
		pauseDuration := time.Since(tl.pauseTime)
		tl.startTime = tl.startTime.Add(pauseDuration)
		tl.state = TimelineRunning
	}
	return tl
}

// Stop stops the timeline.
func (tl *Timeline) Stop() *Timeline {
	tl.state = TimelineCompleted
	return tl
}

// Seek jumps to a specific position on the timeline.
func (tl *Timeline) Seek(position time.Duration) *Timeline {
	tl.playhead = position
	tl.startTime = time.Now().Add(-position)
	tl.updateTweens()
	return tl
}

// SeekLabel jumps to a labeled position.
func (tl *Timeline) SeekLabel(label string) *Timeline {
	if pos, ok := tl.labels[label]; ok {
		return tl.Seek(pos)
	}
	return tl
}

// SeekProgress seeks to a position based on progress (0-1).
func (tl *Timeline) SeekProgress(progress float64) *Timeline {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	return tl.Seek(time.Duration(progress * float64(tl.duration)))
}

// Progress returns the current progress (0-1).
func (tl *Timeline) Progress() float64 {
	if tl.duration == 0 {
		return 0
	}
	return float64(tl.playhead) / float64(tl.duration)
}

// Update advances the timeline.
// Returns true if the timeline is still running.
func (tl *Timeline) Update() bool {
	if tl.state != TimelineRunning {
		return tl.state != TimelineCompleted
	}

	// Calculate elapsed time with time scale
	rawElapsed := time.Since(tl.startTime)
	tl.elapsed = time.Duration(float64(rawElapsed) * tl.timeScale)

	// Calculate playhead position
	if tl.reversed {
		tl.playhead = tl.duration - tl.elapsed
		if tl.playhead < 0 {
			tl.playhead = 0
		}
	} else {
		tl.playhead = tl.elapsed
		if tl.playhead > tl.duration {
			tl.playhead = tl.duration
		}
	}

	// Update all tweens
	tl.updateTweens()

	// Call update callback
	if tl.onUpdate != nil {
		tl.onUpdate(tl.Progress())
	}

	// Check completion
	completed := false
	if tl.reversed {
		completed = tl.playhead <= 0
	} else {
		completed = tl.playhead >= tl.duration
	}

	if completed {
		// Handle repeats
		if tl.repeat != 0 && (tl.repeat < 0 || tl.repeatCount < tl.repeat) {
			tl.repeatCount++
			tl.startTime = time.Now()

			if tl.yoyo {
				tl.reversed = !tl.reversed
			}

			// Reset tweens for next iteration
			for _, entry := range tl.entries {
				entry.tween.Reset()
			}

			return true
		}

		tl.state = TimelineCompleted
		if tl.onComplete != nil {
			tl.onComplete()
		}
		return false
	}

	return true
}

// updateTweens updates all tweens based on current playhead position.
func (tl *Timeline) updateTweens() {
	for _, entry := range tl.entries {
		// Calculate the tween's local time
		localTime := tl.playhead - entry.position

		if localTime < 0 {
			// Tween hasn't started yet
			continue
		}

		// Calculate progress within this tween
		tweenDuration := entry.tween.duration + entry.tween.delay
		if localTime > tweenDuration {
			localTime = tweenDuration
		}

		// Manually update tween's elapsed time
		entry.tween.elapsed = localTime
		if entry.tween.state == TweenPending {
			entry.tween.state = TweenRunning
		}

		// Handle delay
		effectiveLocal := localTime - entry.tween.delay
		if effectiveLocal < 0 {
			continue
		}

		// Calculate progress
		progress := float64(effectiveLocal) / float64(entry.tween.duration)
		if progress > 1 {
			progress = 1
		}

		// Apply easing
		easedProgress := entry.tween.easing(progress)

		// Calculate value
		value := Lerp(entry.tween.from, entry.tween.to, easedProgress)

		// Call update callback
		if entry.tween.onUpdate != nil {
			entry.tween.onUpdate(value)
		}

		// Mark as completed if done
		if progress >= 1 && entry.tween.state != TweenCompleted {
			entry.tween.state = TweenCompleted
			if entry.tween.onComplete != nil {
				entry.tween.onComplete()
			}
		}
	}
}

// IsRunning returns true if the timeline is running.
func (tl *Timeline) IsRunning() bool {
	return tl.state == TimelineRunning
}

// IsCompleted returns true if the timeline has completed.
func (tl *Timeline) IsCompleted() bool {
	return tl.state == TimelineCompleted
}

// State returns the current timeline state.
func (tl *Timeline) State() TimelineState {
	return tl.state
}

// Reverse reverses the timeline playback direction.
func (tl *Timeline) Reverse() *Timeline {
	tl.reversed = !tl.reversed
	return tl
}

// IsReversed returns true if the timeline is playing in reverse.
func (tl *Timeline) IsReversed() bool {
	return tl.reversed
}

// Clear removes all entries from the timeline.
func (tl *Timeline) Clear() *Timeline {
	tl.entries = make([]*TimelineEntry, 0)
	tl.labels = make(map[string]time.Duration)
	tl.duration = 0
	return tl
}

// Kill stops and clears the timeline.
func (tl *Timeline) Kill() *Timeline {
	tl.Stop()
	tl.Clear()
	return tl
}
