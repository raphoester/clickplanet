package catcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "catcher"

type Rules struct {
	Enabled bool

	MinCatches int

	MaxMedian time.Duration

	CertainMedian time.Duration

	Foreign ForeignRules
}

type ForeignRules struct {
	Window        time.Duration
	MinClaims     int
	CertainClaims int
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("minCatches", r.MinCatches),
		detect.Positive("maxMedian", r.MaxMedian),
		detect.Positive("certainMedian", r.CertainMedian),
		detect.AtMost("certainMedian", r.CertainMedian, "maxMedian", r.MaxMedian),
		detect.Positive("foreign.window", r.Foreign.Window),
		detect.NotNegative("foreign.minClaims", r.Foreign.MinClaims),
		detect.NotNegative("foreign.certainClaims", r.Foreign.CertainClaims),
	)
}

type Config struct {
	// Must hold MinCatches boxes at the slowest offer pace, or the rule never fires.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultTrackWindow   = time.Hour
	defaultSweepInterval = time.Minute
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

func (r ForeignRules) kept() int {
	return max(r.MinClaims, r.CertainClaims)
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

func (w *Watchdog) Enabled() bool { return w.rules().Enabled }

func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Caught(scope string, after time.Duration) {
	w.record(scope, outcome{at: w.clock.Now(), caught: true, after: after})
}

func (w *Watchdog) Missed(scope string) {
	w.record(scope, outcome{at: w.clock.Now()})
}

func (w *Watchdog) Foreign(scope string) {
	kept := w.rules().Foreign.kept()
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

	kept := w.rules().MinCatches

	w.mu.Lock()
	defer w.mu.Unlock()

	c := w.callerLocked(scope)
	c.outcomes = append(c.outcomes, o)
	if len(c.outcomes) > kept {
		c.outcomes = append(c.outcomes[:0], c.outcomes[len(c.outcomes)-kept:]...)
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
	rules := w.rules()

	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	catch, catchEvidence := w.catch(c, click.At, rules)
	foreign, foreignEvidence := foreign(c, click.At, rules.Foreign)

	if foreign > catch {
		return foreign, foreignEvidence
	}
	return catch, catchEvidence
}

func (w *Watchdog) catch(c *caller, at time.Time, rules Rules) (detect.Verdict, detect.Evidence) {
	// A rule lowered since keeps more outcomes than it reads: the latest ones count.
	outcomes := c.outcomes[max(0, len(c.outcomes)-rules.MinCatches):]
	if len(outcomes) < rules.MinCatches {
		return detect.Clear, detect.Evidence{}
	}

	cutoff := at.Add(-w.config.TrackWindow)

	delays := make([]time.Duration, 0, len(outcomes))
	for _, o := range outcomes {
		if !o.caught || o.at.Before(cutoff) {
			return detect.Clear, detect.Evidence{}
		}
		delays = append(delays, o.after)
	}

	median, _ := detect.Spread(delays)
	if median > rules.MaxMedian {
		return detect.Clear, detect.Evidence{}
	}

	verdict := detect.Suspect
	if median <= rules.CertainMedian {
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

func foreign(c *caller, at time.Time, rules ForeignRules) (detect.Verdict, detect.Evidence) {
	cutoff := at.Add(-rules.Window)
	claims := 0
	for _, claimed := range c.foreign {
		if claimed.After(cutoff) {
			claims++
		}
	}

	verdict := detect.Clear
	switch {
	case rules.CertainClaims > 0 && claims >= rules.CertainClaims:
		verdict = detect.Certain
	case rules.MinClaims > 0 && claims >= rules.MinClaims:
		verdict = detect.Suspect
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "foreign",
		Fields: []detect.Field{
			{Key: "claims", Value: claims},
			{Key: "within", Value: rules.Window},
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

	cutoff := now.Add(-max(w.config.TrackWindow, w.rules().Foreign.Window))
	for scope, c := range w.callers {
		if c.lastSeen().Before(cutoff) {
			delete(w.callers, scope)
		}
	}
}
