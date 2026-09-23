// Package metronome watches for the caller that never varies and never stops.
// A script tuned to sit just under the throttle spends hours at one tempo with
// no pauses in it. Tempo alone says nothing — a player can click fast. What no
// hand produces is the same gap, again and again, for hours, without once
// looking away. A random sleep is a clock too: its gaps sit evenly where a hand's
// lean long. And a timer keeps its beat through a pause: a hand that rests
// comes back on no beat at all. The gaps are between clicks tried, not clicks
// accepted.
package metronome

import (
	"context"
	"math"
	"sort"
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

	Shape ShapeConfig

	Clock ClockConfig

	// TrackWindow is how long a silent caller is remembered.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

// ShapeConfig bounds the skew of the gaps, (p90 + p10 - 2*p50) / (p90 - p10): near 0 for a random sleep, towards 1 for a hand.
type ShapeConfig struct {
	// MaxGap is the longest gap kept as a sample; a longer one is skipped and ends nothing.
	MaxGap time.Duration

	// A nil skew never reads its level; a pointer because 0 is a skew.
	Clicks        int
	MaxSkew       *float64
	CertainClicks int
	CertainSkew   *float64
}

// ClockConfig bounds how closely the clicks tried keep one beat: the length of
// the mean of the unit vectors at each try's place on a Period-long circle,
// 1 when every try lands at the same point of the beat and near 0 for a hand.
// It reads absolute times, not gaps, so a pause breaks nothing: a timer that
// waits for the bucket comes back on its beat, and a hand does not.
type ClockConfig struct {
	// Period is the beat. A timer in a hidden tab fires on whole seconds, and
	// any whole number of seconds is on the same beat.
	Period time.Duration

	// A nil coherence never reads its level; a pointer because 0 is a coherence.
	Clicks        int
	MinCoherence  *float64
	CertainClicks int
	// CertainFor is how long the CertainClicks must span: a person tapping to a
	// song keeps a beat for a song, not for half an hour.
	CertainFor       time.Duration
	CertainCoherence *float64
}

const (
	defaultMaxGap        = 3 * time.Second
	defaultMaxSpread     = 120 * time.Millisecond
	defaultMinClicks     = 120
	defaultCertainFor    = 30 * time.Minute
	defaultCertainClicks = 900
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	defaultShapeMaxGap        = 10 * time.Second
	defaultShapeClicks        = 500
	defaultShapeCertainClicks = 1000

	defaultClockPeriod        = time.Second
	defaultClockClicks        = 120
	defaultClockCertainClicks = 600
	defaultClockCertainFor    = 30 * time.Minute

	maxSamples      = 1024
	maxShapeSamples = 2048
	maxClockSamples = 2048
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
	c.Shape = c.Shape.withDefaults()
	c.Clock = c.Clock.withDefaults()
	return c
}

func (c ClockConfig) withDefaults() ClockConfig {
	if c.Period <= 0 {
		c.Period = defaultClockPeriod
	}
	if c.Clicks <= 0 {
		c.Clicks = defaultClockClicks
	}
	if c.Clicks > maxClockSamples {
		c.Clicks = maxClockSamples
	}
	if c.CertainClicks <= 0 {
		c.CertainClicks = defaultClockCertainClicks
	}
	if c.CertainClicks < c.Clicks {
		c.CertainClicks = c.Clicks
	}
	if c.CertainClicks > maxClockSamples {
		c.CertainClicks = maxClockSamples
	}
	if c.CertainFor <= 0 {
		c.CertainFor = defaultClockCertainFor
	}
	return c
}

// kept is how many tries the clock holds: none while neither level is set.
func (c ClockConfig) kept() int {
	switch {
	case c.CertainCoherence != nil:
		return c.CertainClicks
	case c.MinCoherence != nil:
		return c.Clicks
	default:
		return 0
	}
}

func (c ShapeConfig) withDefaults() ShapeConfig {
	if c.MaxGap <= 0 {
		c.MaxGap = defaultShapeMaxGap
	}
	if c.Clicks <= 0 {
		c.Clicks = defaultShapeClicks
	}
	if c.Clicks > maxShapeSamples {
		c.Clicks = maxShapeSamples
	}
	if c.CertainClicks <= 0 {
		c.CertainClicks = defaultShapeCertainClicks
	}
	if c.CertainClicks < c.Clicks {
		c.CertainClicks = c.Clicks
	}
	if c.CertainClicks > maxShapeSamples {
		c.CertainClicks = maxShapeSamples
	}
	return c
}

// New takes onSkew and onCoherence, called each sweep with every caller's reading once its window is full.
func New(config Config, clock cptime.Clock, onSkew, onCoherence func(float64)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:      config.withDefaults(),
		clock:       clock,
		onSkew:      onSkew,
		onCoherence: onCoherence,
		callers:     make(map[string]*caller),
	}
}

type Watchdog struct {
	config      Config
	clock       cptime.Clock
	onSkew      func(float64)
	onCoherence func(float64)

	mu      sync.Mutex
	callers map[string]*caller
	outage  detect.Outage
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

	// shape is not cleared by a break: the bot it exists for pauses between bursts.
	shape []time.Duration

	// tries is when each of the last clicks was tried, for the clock. Not cleared by a break either.
	tries []time.Time
}

func (w *Watchdog) Name() string { return Name }

// Committed is nothing to this watchdog. What the map did with a click has no
// bearing on when the next one arrived.
func (w *Watchdog) Committed(detect.Click) {}

// Attempted times the run: the throttle's survivors no longer carry the loop's gaps.
func (w *Watchdog) Attempted(click detect.Click) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		c = &caller{}
		w.callers[click.Scope] = c
		c.restart(click.At)
		w.tried(c, click.At)
		return
	}

	w.tried(c, click.At)

	across := w.outage.Across(c.lastSeen, click.At)
	gap := w.outage.Gap(c.lastSeen, click.At)
	c.lastSeen = click.At

	if !across && gap >= 0 && gap <= w.config.Shape.MaxGap {
		c.shape = append(c.shape, gap)
		if capacity := w.config.Shape.CertainClicks; len(c.shape) > capacity {
			c.shape = append(c.shape[:0], c.shape[len(c.shape)-capacity:]...)
		}
	}

	if gap < 0 || gap > w.config.MaxGap {
		c.restart(click.At)
		return
	}

	c.runClicks++

	// Stitched across a restart, not measured: neither a break nor a sample, and the outage is not time sustained.
	if across {
		c.runStart = c.runStart.Add(w.outage.Length())
		return
	}

	c.gaps = append(c.gaps, gap)
	if capacity := w.capacity(); len(c.gaps) > capacity {
		c.gaps = append(c.gaps[:0], c.gaps[len(c.gaps)-capacity:]...)
	}
}

// tried keeps the time of a try for the clock. A restart is no reason to skip one: it reads when, not how long after.
func (w *Watchdog) tried(c *caller, at time.Time) {
	kept := w.config.Clock.kept()
	if kept == 0 {
		return
	}

	c.tries = append(c.tries, at)
	if len(c.tries) > kept {
		c.tries = append(c.tries[:0], c.tries[len(c.tries)-kept:]...)
	}
}

// Watch judges the run Attempted has timed so far; it records nothing itself.
func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	verdict, evidence := w.cadence(c)

	// The stronger level is reported, the earlier rule on a tie.
	if shape, shapeEvidence := w.shape(c); shape > verdict {
		verdict, evidence = shape, shapeEvidence
	}
	if timer, timerEvidence := w.timer(c); timer > verdict {
		verdict, evidence = timer, timerEvidence
	}

	return verdict, evidence
}

func (w *Watchdog) cadence(c *caller) (detect.Verdict, detect.Evidence) {
	if c.runClicks < w.config.MinClicks {
		return detect.Clear, detect.Evidence{}
	}

	median, spread := detect.Spread(append([]time.Duration(nil), c.gaps...))
	if spread > w.config.MaxSpread {
		return detect.Clear, detect.Evidence{}
	}

	sustained := c.lastSeen.Sub(c.runStart)

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

func (w *Watchdog) shape(c *caller) (detect.Verdict, detect.Evidence) {
	config := w.config.Shape

	if config.CertainSkew != nil {
		if s, ok := skewOf(c.shape, config.CertainClicks); ok && s.skew <= *config.CertainSkew {
			return detect.Certain, s.evidence()
		}
	}

	if config.MaxSkew != nil {
		if s, ok := skewOf(c.shape, config.Clicks); ok && s.skew <= *config.MaxSkew {
			return detect.Suspect, s.evidence()
		}
	}

	return detect.Clear, detect.Evidence{}
}

type skewed struct {
	skew          float64
	p10, p50, p90 time.Duration
	clicks        int
}

// skewOf reads nothing from gaps that do not spread: a gap that never varies is cadence's.
func skewOf(gaps []time.Duration, n int) (skewed, bool) {
	if len(gaps) < n {
		return skewed{}, false
	}

	sorted := append([]time.Duration(nil), gaps[len(gaps)-n:]...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	s := skewed{
		p10:    detect.Quantile(sorted, 0.1),
		p50:    detect.Quantile(sorted, 0.5),
		p90:    detect.Quantile(sorted, 0.9),
		clicks: n,
	}
	if s.p90 <= s.p10 {
		return skewed{}, false
	}

	s.skew = float64(s.p90+s.p10-2*s.p50) / float64(s.p90-s.p10)
	return s, true
}

func (s skewed) evidence() detect.Evidence {
	return detect.Evidence{
		Rule: "shape",
		Fields: []detect.Field{
			{Key: "skew", Value: math.Round(s.skew*100) / 100},
			{Key: "p10", Value: s.p10},
			{Key: "median", Value: s.p50},
			{Key: "p90", Value: s.p90},
			{Key: "clicks", Value: s.clicks},
		},
	}
}

func (w *Watchdog) timer(c *caller) (detect.Verdict, detect.Evidence) {
	config := w.config.Clock

	if config.CertainCoherence != nil {
		if b, ok := beatOf(c.tries, config.CertainClicks, config.Period); ok &&
			b.coherence >= *config.CertainCoherence && b.span >= config.CertainFor {
			return detect.Certain, b.evidence()
		}
	}

	if config.MinCoherence != nil {
		if b, ok := beatOf(c.tries, config.Clicks, config.Period); ok && b.coherence >= *config.MinCoherence {
			return detect.Suspect, b.evidence()
		}
	}

	return detect.Clear, detect.Evidence{}
}

type beat struct {
	coherence float64
	period    time.Duration
	offset    time.Duration // where on the beat the tries land
	span      time.Duration // from the first try counted to the last
	clicks    int
}

// beatOf reads the last n tries on a circle one period long; false until there are n.
func beatOf(tries []time.Time, n int, period time.Duration) (beat, bool) {
	if len(tries) < n || n == 0 {
		return beat{}, false
	}

	window := tries[len(tries)-n:]

	var x, y float64
	for _, at := range window {
		angle := 2 * math.Pi * float64(at.UnixNano()%int64(period)) / float64(period)
		x += math.Cos(angle)
		y += math.Sin(angle)
	}
	x /= float64(n)
	y /= float64(n)

	angle := math.Atan2(y, x)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	return beat{
		coherence: math.Hypot(x, y),
		period:    period,
		offset:    time.Duration(angle / (2 * math.Pi) * float64(period)).Round(time.Millisecond),
		span:      window[n-1].Sub(window[0]),
		clicks:    n,
	}, true
}

func (b beat) evidence() detect.Evidence {
	return detect.Evidence{
		Rule: "clock",
		Fields: []detect.Field{
			{Key: "coherence", Value: math.Round(b.coherence*100) / 100},
			{Key: "period", Value: b.period},
			{Key: "offset", Value: b.offset},
			{Key: "clicks", Value: b.clicks},
			{Key: "over", Value: b.span.Round(time.Second)},
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
	now := w.clock.Now()

	var (
		skews      []float64
		coherences []float64
	)

	w.mu.Lock()

	cutoff := now.Add(-w.config.TrackWindow)
	for scope, c := range w.callers {
		if c.lastSeen.Before(cutoff) {
			delete(w.callers, scope)
			continue
		}
		if s, ok := skewOf(c.shape, w.config.Shape.Clicks); ok {
			skews = append(skews, s.skew)
		}
		if b, ok := beatOf(c.tries, w.config.Clock.Clicks, w.config.Clock.Period); ok {
			coherences = append(coherences, b.coherence)
		}
	}

	w.mu.Unlock()

	for _, skew := range skews {
		w.onSkew(skew)
	}
	for _, coherence := range coherences {
		w.onCoherence(coherence)
	}
}
