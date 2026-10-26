package sequencer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "sequencer"

type Rules struct {
	MinSteps int
	MinShare float64

	CertainSteps int
	CertainShare float64
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("minSteps", r.MinSteps),
		detect.Positive("minShare", r.MinShare),
		detect.Share("minShare", r.MinShare),
		detect.AtLeast("certainSteps", r.CertainSteps, "minSteps", r.MinSteps),
		detect.AtLeast("certainShare", r.CertainShare, "minShare", r.MinShare),
		detect.Share("certainShare", r.CertainShare),
	)
}

type Config struct {
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	maxSamples = 4096
)

func (c Config) withDefaults() Config {
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, rules func() Rules, clock cptime.Clock) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:  config.withDefaults(),
		rules:   rules,
		clock:   clock,
		callers: make(map[string]*caller),
	}
}

type Watchdog struct {
	config Config
	rules  func() Rules
	clock  cptime.Clock

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

func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	rules := w.rules()

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
		}, w.config.TrackWindow, capacity(rules))
	}

	c.lastTile = click.Tile
	c.seen = true

	c.prune(click.At.Add(-w.config.TrackWindow))

	if len(c.steps) < rules.MinSteps {
		return detect.Clear, detect.Evidence{}
	}

	stride, share := c.stride()

	if stride == 0 || share < rules.MinShare {
		return detect.Clear, detect.Evidence{}
	}

	verdict := detect.Suspect
	if len(c.steps) >= rules.CertainSteps && share >= rules.CertainShare {
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

func capacity(rules Rules) int {
	return min(rules.CertainSteps, maxSamples)
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
	now := w.clock.Now()

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
