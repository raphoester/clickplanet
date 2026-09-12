// Package metronome watches for the caller that never varies and never stops.
// A script tuned to sit just under the throttle spends hours at one tempo with
// no pauses in it. Tempo alone says nothing — a player can click fast, and a
// caller pushing past the throttle gets its surviving clicks handed back at
// exactly the refill rate. What no hand produces is the same gap, again and
// again, for hours, without once looking away.
package metronome

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "metronome"

type Config struct {
	// MaxGap is the longest pause a run survives. Anything longer ends the run
	// and the evidence starts again from nothing, which is the whole point: a
	// person stops to look at the map, and a loop does not.
	MaxGap time.Duration

	// MaxSpread is the p90-p10 of the gaps inside a run. This is the bound that
	// does the work. The median is deliberately not bounded at all: the claim
	// is not that the caller is fast, it is that the caller is a clock.
	MaxSpread time.Duration

	// MinClicks is how long an unbroken run must be before it reads as Suspect.
	MinClicks int

	// CertainFor and CertainClicks are the same run held past anything a person
	// sustains. Half an hour without once pausing longer than MaxGap is not
	// somebody who is very keen.
	CertainFor    time.Duration
	CertainClicks int

	// TrackWindow is how long a silent caller is remembered.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMaxGap        = 3 * time.Second
	defaultMaxSpread     = 120 * time.Millisecond
	defaultMinClicks     = 120
	defaultCertainFor    = 30 * time.Minute
	defaultCertainClicks = 900
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	maxSamples = 1024
)

func (c Config) withDefaults() Config {
	if c.MaxGap <= 0 {
		c.MaxGap = defaultMaxGap
	}
	if c.MaxSpread <= 0 {
		c.MaxSpread = defaultMaxSpread
	}
	if c.MinClicks <= 0 {
		c.MinClicks = defaultMinClicks
	}
	if c.CertainFor <= 0 {
		c.CertainFor = defaultCertainFor
	}
	if c.CertainClicks <= 0 {
		c.CertainClicks = defaultCertainClicks
	}
	if c.CertainClicks < c.MinClicks {
		c.CertainClicks = c.MinClicks
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, timeProvider cptime.Provider) *Watchdog {
	if timeProvider == nil {
		timeProvider = cptime.ActualProvider{}
	}

	return &Watchdog{
		config:       config.withDefaults(),
		timeProvider: timeProvider,
		callers:      make(map[string]*caller),
	}
}

type Watchdog struct {
	config       Config
	timeProvider cptime.Provider

	mu      sync.Mutex
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

// caller holds one run. Only the gaps needed for a spread are kept: how long the
// run has lasted is two timestamps, not a list, and a run that breaks is thrown
// away rather than pruned.
type caller struct {
	lastSeen time.Time

	runStart  time.Time
	runClicks int

	gaps []time.Duration
}

func (w *Watchdog) Name() string { return Name }

// Committed is nothing to this watchdog. What the map did with a click has no
// bearing on when the next one arrived.
func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		c = &caller{}
		w.callers[click.Scope] = c
		c.restart(click.At)
		return detect.Clear, detect.Evidence{}
	}

	gap := click.At.Sub(c.lastSeen)
	c.lastSeen = click.At

	if gap < 0 || gap > w.config.MaxGap {
		c.restart(click.At)
		return detect.Clear, detect.Evidence{}
	}

	c.runClicks++
	c.gaps = append(c.gaps, gap)
	if capacity := w.capacity(); len(c.gaps) > capacity {
		c.gaps = append(c.gaps[:0], c.gaps[len(c.gaps)-capacity:]...)
	}

	if c.runClicks < w.config.MinClicks {
		return detect.Clear, detect.Evidence{}
	}

	median, spread := detect.Spread(append([]time.Duration(nil), c.gaps...))
	if spread > w.config.MaxSpread {
		return detect.Clear, detect.Evidence{}
	}

	sustained := click.At.Sub(c.runStart)

	verdict := detect.Suspect
	if sustained >= w.config.CertainFor && c.runClicks >= w.config.CertainClicks {
		verdict = detect.Certain
	}

	return verdict, detect.Evidence{
		Rule: "cadence",
		Fields: []detect.Field{
			{Key: "spread", Value: spread},
			{Key: "median", Value: median},
			{Key: "clicks", Value: c.runClicks},
			{Key: "sustained", Value: sustained},
		},
	}
}

func (w *Watchdog) capacity() int {
	if w.config.MinClicks > maxSamples {
		return maxSamples
	}
	return w.config.MinClicks
}

func (c *caller) restart(at time.Time) {
	c.lastSeen = at
	c.runStart = at
	c.runClicks = 1
	c.gaps = c.gaps[:0]
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
		if c.lastSeen.Before(cutoff) {
			delete(w.callers, scope)
		}
	}
}
