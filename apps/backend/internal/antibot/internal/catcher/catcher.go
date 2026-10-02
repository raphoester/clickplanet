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
	MinCatches int

	MaxMedian time.Duration

	CertainMedian time.Duration

	// Must hold MinCatches boxes at the slowest offer pace, or the rule never fires.
	TrackWindow time.Duration

	Foreign ForeignConfig

	SweepInterval time.Duration
}

type ForeignConfig struct {
	Window        time.Duration
	MinClaims     int
	CertainClaims int
}

const (
	defaultMinCatches    = 5
	defaultMaxMedian     = 3 * time.Second
	defaultCertainMedian = 1500 * time.Millisecond
	defaultTrackWindow   = time.Hour
	defaultSweepInterval = time.Minute

	defaultForeignWindow = time.Hour
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
	if c.Foreign.Window <= 0 {
		c.Foreign.Window = defaultForeignWindow
	}
	if c.Foreign.CertainClaims > 0 && c.Foreign.CertainClaims < c.Foreign.MinClaims {
		c.Foreign.CertainClaims = c.Foreign.MinClaims
	}
	return c
}

func (c ForeignConfig) kept() int {
	return max(c.MinClaims, c.CertainClaims)
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

type caller struct {
	outcomes []outcome
	foreign  []time.Time
}

func (c *caller) lastSeen() time.Time {
	var last time.Time
	if len(c.outcomes) > 0 {
		last = c.outcomes[len(c.outcomes)-1].at
	}
	if len(c.foreign) > 0 && c.foreign[len(c.foreign)-1].After(last) {
		last = c.foreign[len(c.foreign)-1]
	}
	return last
}

type outcome struct {
	at     time.Time
	caught bool
	after  time.Duration
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Caught(scope string, after time.Duration) {
	w.record(scope, outcome{at: w.clock.Now(), caught: true, after: after})
}

func (w *Watchdog) Missed(scope string) {
	w.record(scope, outcome{at: w.clock.Now()})
}

func (w *Watchdog) Foreign(scope string) {
	kept := w.config.Foreign.kept()
	if scope == "" || kept == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	c := w.callerLocked(scope)
	c.foreign = append(c.foreign, w.clock.Now())
	if len(c.foreign) > kept {
		c.foreign = append(c.foreign[:0], c.foreign[len(c.foreign)-kept:]...)
	}
}

func (w *Watchdog) record(scope string, o outcome) {
	if scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	c := w.callerLocked(scope)
	c.outcomes = append(c.outcomes, o)
	if len(c.outcomes) > w.config.MinCatches {
		c.outcomes = append(c.outcomes[:0], c.outcomes[len(c.outcomes)-w.config.MinCatches:]...)
	}
}

func (w *Watchdog) callerLocked(scope string) *caller {
	c, ok := w.callers[scope]
	if !ok {
		c = &caller{}
		w.callers[scope] = c
	}
	return c
}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	catch, catchEvidence := w.catch(c, click.At)
	foreign, foreignEvidence := w.foreign(c, click.At)

	if foreign > catch {
		return foreign, foreignEvidence
	}
	return catch, catchEvidence
}

func (w *Watchdog) catch(c *caller, at time.Time) (detect.Verdict, detect.Evidence) {
	if len(c.outcomes) < w.config.MinCatches {
		return detect.Clear, detect.Evidence{}
	}

	cutoff := at.Add(-w.config.TrackWindow)

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

func (w *Watchdog) foreign(c *caller, at time.Time) (detect.Verdict, detect.Evidence) {
	config := w.config.Foreign

	cutoff := at.Add(-config.Window)
	claims := 0
	for _, claimed := range c.foreign {
		if claimed.After(cutoff) {
			claims++
		}
	}

	verdict := detect.Clear
	switch {
	case config.CertainClaims > 0 && claims >= config.CertainClaims:
		verdict = detect.Certain
	case config.MinClaims > 0 && claims >= config.MinClaims:
		verdict = detect.Suspect
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "foreign",
		Fields: []detect.Field{
			{Key: "claims", Value: claims},
			{Key: "within", Value: config.Window},
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

	cutoff := now.Add(-max(w.config.TrackWindow, w.config.Foreign.Window))
	for scope, c := range w.callers {
		if c.lastSeen().Before(cutoff) {
			delete(w.callers, scope)
		}
	}
}
