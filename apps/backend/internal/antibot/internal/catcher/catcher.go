// Package catcher watches for the caller that catches every bonus box, at once.
// A box is sent down one caller's stream and flies a slow orbit that is rarely
// in view: a person has to zoom out to orbit height and often drag the globe
// round before they can click it, and some boxes go by unseen. A script reads
// the offer off the stream and claims it before the box has left its spawn.
// Speed alone is not the claim — a player already zoomed out gets lucky — and
// catching every box is not either. Both, box after box, is.
package catcher

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "catcher"

type Config struct {
	// MinCatches is how many boxes in a row, the last ones offered, must all
	// have been caught before anything is said. A single miss among them clears
	// the caller: a person misses boxes and a script does not.
	MinCatches int

	// MaxMedian is the median delay from offer to claim, over those catches,
	// that reads as Suspect.
	MaxMedian time.Duration

	// CertainMedian is the same median at a speed no person manages box after
	// box: find the box, reach it, click it.
	CertainMedian time.Duration

	// TrackWindow is how long an outcome counts. It must hold MinCatches boxes
	// at the slowest pace they are offered, or the rule can never fire.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMinCatches    = 5
	defaultMaxMedian     = 3 * time.Second
	defaultCertainMedian = 1500 * time.Millisecond
	defaultTrackWindow   = 30 * time.Minute
	defaultSweepInterval = time.Minute
)

func (c Config) withDefaults() Config {
	if c.MinCatches <= 0 {
		c.MinCatches = defaultMinCatches
	}
	if c.MaxMedian <= 0 {
		c.MaxMedian = defaultMaxMedian
	}
	if c.CertainMedian <= 0 {
		c.CertainMedian = defaultCertainMedian
	}
	if c.CertainMedian > c.MaxMedian {
		c.CertainMedian = c.MaxMedian
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:  config.withDefaults(),
		clock:   clock,
		callers: make(map[string]*caller),
	}
}

type Watchdog struct {
	config Config
	clock  cptime.Clock

	mu      sync.Mutex
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

// caller holds the last MinCatches boxes offered, oldest first.
type caller struct {
	outcomes []outcome
}

type outcome struct {
	at     time.Time
	caught bool
	after  time.Duration
}

func (w *Watchdog) Name() string { return Name }

// Committed is nothing to this watchdog: it reads boxes, not tiles.
func (w *Watchdog) Committed(detect.Click) {}

// Caught records a box claimed after the delay since it was offered.
func (w *Watchdog) Caught(scope string, after time.Duration) {
	w.record(scope, outcome{at: w.clock.Now(), caught: true, after: after})
}

// Missed records a box offered and never claimed.
func (w *Watchdog) Missed(scope string) {
	w.record(scope, outcome{at: w.clock.Now()})
}

func (w *Watchdog) record(scope string, o outcome) {
	if scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[scope]
	if !ok {
		c = &caller{}
		w.callers[scope] = c
	}

	c.outcomes = append(c.outcomes, o)
	if len(c.outcomes) > w.config.MinCatches {
		c.outcomes = append(c.outcomes[:0], c.outcomes[len(c.outcomes)-w.config.MinCatches:]...)
	}
}

// Watch answers from the boxes already caught, not from the click: a box is
// claimed between clicks, and the jury only asks on a click.
func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok || len(c.outcomes) < w.config.MinCatches {
		return detect.Clear, detect.Evidence{}
	}

	cutoff := click.At.Add(-w.config.TrackWindow)

	delays := make([]time.Duration, 0, len(c.outcomes))
	for _, o := range c.outcomes {
		if !o.caught || o.at.Before(cutoff) {
			return detect.Clear, detect.Evidence{}
		}
		delays = append(delays, o.after)
	}

	median, _ := detect.Spread(delays)
	if median > w.config.MaxMedian {
		return detect.Clear, detect.Evidence{}
	}

	verdict := detect.Suspect
	if median <= w.config.CertainMedian {
		verdict = detect.Certain
	}

	return verdict, detect.Evidence{
		Rule: "catch",
		Fields: []detect.Field{
			{Key: "median", Value: median},
			{Key: "slowest", Value: delays[len(delays)-1]},
			{Key: "caught", Value: len(delays)},
		},
	}
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
		if c.outcomes[len(c.outcomes)-1].at.Before(cutoff) {
			delete(w.callers, scope)
		}
	}
}
