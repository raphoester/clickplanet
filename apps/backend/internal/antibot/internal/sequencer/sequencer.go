// Package sequencer watches for the caller walking the tile ids rather than the
// map: 1, 2, 3, 4, and on until a continent is painted. Tile ids come from the
// icosahedron's vertex order, not from a grid of latitudes, so a hand filling in
// a shape does not produce a constant step between one click and the next. A
// loop over an integer does nothing else.
package sequencer

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/xtime"
)

const Name = "sequencer"

type Config struct {
	// MinSteps is how many clicks in a row must be in hand before anything is
	// said, and MinShare how many of them must sit at the same step for the
	// caller to read as Suspect. A hand wanders: it clicks a neighbour, then
	// back, then three tiles over.
	MinSteps int
	MinShare float64

	// CertainSteps and CertainShare are the same reading held for long enough
	// that nothing else explains it. Four hundred clicks at a constant step is
	// not a player being tidy.
	CertainSteps int
	CertainShare float64

	// TrackWindow is how far back steps count.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMinSteps      = 40
	defaultMinShare      = 0.75
	defaultCertainSteps  = 200
	defaultCertainShare  = 0.95
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	maxSamples = 4096
)

func (c Config) withDefaults() Config {
	if c.MinSteps <= 0 {
		c.MinSteps = defaultMinSteps
	}
	if c.MinShare <= 0 {
		c.MinShare = defaultMinShare
	}
	if c.CertainSteps <= 0 {
		c.CertainSteps = defaultCertainSteps
	}
	if c.CertainShare <= 0 {
		c.CertainShare = defaultCertainShare
	}
	if c.CertainSteps < c.MinSteps {
		c.CertainSteps = c.MinSteps
	}
	if c.CertainShare < c.MinShare {
		c.CertainShare = c.MinShare
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, timeProvider xtime.Provider) *Watchdog {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	return &Watchdog{
		config:       config.withDefaults(),
		timeProvider: timeProvider,
		callers:      make(map[string]*caller),
	}
}

type Watchdog struct {
	config       Config
	timeProvider xtime.Provider

	mu      sync.Mutex
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

type caller struct {
	lastTile uint32
	seen     bool

	steps []step
}

type step struct {
	at   time.Time
	size int64
}

func (w *Watchdog) Name() string { return Name }

// Committed is nothing to this watchdog. A bot sweeping ids walks over tiles it
// already owns and over ids the handler refuses, and both are part of the walk.
func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		c = &caller{}
		w.callers[click.Scope] = c
	}

	if c.seen {
		c.record(step{
			at:   click.At,
			size: int64(click.Tile) - int64(c.lastTile),
		}, w.config.TrackWindow, w.capacity())
	}

	c.lastTile = click.Tile
	c.seen = true

	c.prune(click.At.Add(-w.config.TrackWindow))

	if len(c.steps) < w.config.MinSteps {
		return detect.Clear, detect.Evidence{}
	}

	stride, share := c.stride()

	// A caller that mostly re-clicks one tile has a modal step of zero. That is
	// somebody leaning on a tile, which is the throttle's problem and the
	// retaker's, not a walk.
	if stride == 0 || share < w.config.MinShare {
		return detect.Clear, detect.Evidence{}
	}

	verdict := detect.Suspect
	if len(c.steps) >= w.config.CertainSteps && share >= w.config.CertainShare {
		verdict = detect.Certain
	}

	return verdict, detect.Evidence{
		Rule: "stride",
		Fields: []detect.Field{
			{Key: "stride", Value: stride},
			{Key: "share", Value: share},
			{Key: "steps", Value: len(c.steps)},
		},
	}
}

func (w *Watchdog) capacity() int {
	if w.config.CertainSteps > maxSamples {
		return maxSamples
	}
	return w.config.CertainSteps
}

func (c *caller) record(s step, window time.Duration, capacity int) {
	c.prune(s.at.Add(-window))

	c.steps = append(c.steps, s)
	if len(c.steps) > capacity {
		c.steps = append(c.steps[:0], c.steps[len(c.steps)-capacity:]...)
	}
}

func (c *caller) prune(cutoff time.Time) {
	kept := c.steps[:0]
	for _, s := range c.steps {
		if s.at.After(cutoff) {
			kept = append(kept, s)
		}
	}
	c.steps = kept
}

// stride is the step the caller took most often, and the share of its steps
// that were that one.
func (c *caller) stride() (int64, float64) {
	counts := make(map[int64]int, len(c.steps))
	for _, s := range c.steps {
		counts[s.size]++
	}

	var (
		stride int64
		count  int
	)

	for size, n := range counts {
		if n > count || (n == count && size < stride) {
			stride, count = size, n
		}
	}

	return stride, float64(count) / float64(len(c.steps))
}

func (w *Watchdog) Run(ctx context.Context) {
	ticker := time.NewTicker(w.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watchdog) sweep() {
	now := w.timeProvider.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := now.Add(-w.config.TrackWindow)
	for scope, c := range w.callers {
		c.prune(cutoff)
		if len(c.steps) == 0 {
			delete(w.callers, scope)
		}
	}
}
