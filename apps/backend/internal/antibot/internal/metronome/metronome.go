// Package metronome watches for the caller that never varies and never stops.
// A script tuned to sit just under the throttle spends hours at one tempo with
// no pauses in it. Tempo alone says nothing — a player can click fast. What no
// hand produces is the same gap, again and again, for hours, without once
// looking away. A random sleep is a clock too: its gaps sit evenly where a hand's
// lean long. The gaps are between clicks tried, not clicks accepted.
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

	Stamina StaminaConfig

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

// StaminaConfig bounds the time in Window one payer spent at least Clicks a Slice; zero never reads a level.
type StaminaConfig struct {
	Slice       time.Duration
	Clicks      int
	Window      time.Duration
	MinBusy     time.Duration
	CertainBusy time.Duration
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

	defaultStaminaSlice  = 10 * time.Minute
	defaultStaminaClicks = 40
	defaultStaminaWindow = 6 * time.Hour

	maxSamples      = 1024
	maxShapeSamples = 2048
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
	c.Stamina = c.Stamina.withDefaults()
	return c
}

func (c StaminaConfig) withDefaults() StaminaConfig {
	if c.Slice <= 0 {
		c.Slice = defaultStaminaSlice
	}
	if c.Clicks <= 0 {
		c.Clicks = defaultStaminaClicks
	}
	if c.Window <= 0 {
		c.Window = defaultStaminaWindow
	}
	if c.Window < c.Slice {
		c.Window = c.Slice
	}
	return c
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

func New(config Config, clock cptime.Clock, onSkew func(skew float64), onBusy func(busy time.Duration)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:   config.withDefaults(),
		clock:    clock,
		onSkew:   onSkew,
		onBusy:   onBusy,
		callers:  make(map[string]*caller),
		spenders: make(map[string]*spender),
	}
}

type Watchdog struct {
	config Config
	clock  cptime.Clock
	onSkew func(float64)
	onBusy func(time.Duration)

	mu       sync.Mutex
	callers  map[string]*caller
	spenders map[string]*spender
	outage   detect.Outage
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
}

// spender is keyed on the payer, not the scope: the tokens it counts are the account's.
type spender struct {
	lastSeen time.Time
	slices   []slice // oldest first, only the ones with a click
}

type slice struct {
	index  int64
	clicks int
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
		return
	}

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

// Watch judges the run Attempted has timed so far, and counts the click the throttle let through against its payer.
func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	payer := payerOf(click)
	w.spend(payer, click.At)
	stamina, staminaEvidence := w.stamina(w.spenders[payer])

	c, ok := w.callers[click.Scope]
	if !ok {
		return stamina, staminaEvidence
	}

	// The stronger level; on a tie, cadence, then shape.
	verdict, evidence := w.cadence(c)
	if shape, shapeEvidence := w.shape(c); shape > verdict {
		verdict, evidence = shape, shapeEvidence
	}
	if stamina > verdict {
		verdict, evidence = stamina, staminaEvidence
	}
	return verdict, evidence
}

// payerOf names the bucket the click spent from, as the throttle keys it.
func payerOf(click detect.Click) string {
	if click.Account != "" {
		return "account:" + click.Account
	}
	return "scope:" + click.Scope
}

func (w *Watchdog) spend(payer string, at time.Time) {
	s, ok := w.spenders[payer]
	if !ok {
		s = &spender{}
		w.spenders[payer] = s
	}
	s.lastSeen = at

	index := w.sliceOf(at)
	// A click stamped just before the last one's slice began is counted in it: they are milliseconds apart.
	if last := len(s.slices) - 1; last >= 0 && index <= s.slices[last].index {
		s.slices[last].clicks++
	} else {
		s.slices = append(s.slices, slice{index: index, clicks: 1})
	}

	s.forget(w.firstSliceOf(index))
}

func (w *Watchdog) stamina(s *spender) (detect.Verdict, detect.Evidence) {
	config := w.config.Stamina

	busySlices, clicks := s.tally(config.Clicks)
	busy := time.Duration(busySlices) * config.Slice

	var verdict detect.Verdict
	switch {
	case config.CertainBusy > 0 && busy >= config.CertainBusy:
		verdict = detect.Certain
	case config.MinBusy > 0 && busy >= config.MinBusy:
		verdict = detect.Suspect
	default:
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "stamina",
		Fields: []detect.Field{
			{Key: "busy", Value: busy},
			{Key: "window", Value: config.Window},
			{Key: "clicks", Value: clicks},
		},
	}
}

func (w *Watchdog) sliceOf(at time.Time) int64 {
	return at.UnixNano() / int64(w.config.Stamina.Slice)
}

// firstSliceOf is the oldest slice still inside the window that ends with index.
func (w *Watchdog) firstSliceOf(index int64) int64 {
	return index - int64(w.config.Stamina.Window/w.config.Stamina.Slice) + 1
}

// tally is how many slices hold at least atLeast clicks, and how many clicks all of them hold.
func (s *spender) tally(atLeast int) (busy, clicks int) {
	for _, counted := range s.slices {
		if counted.clicks >= atLeast {
			busy++
		}
		clicks += counted.clicks
	}
	return busy, clicks
}

func (s *spender) forget(before int64) {
	kept := 0
	for kept < len(s.slices) && s.slices[kept].index < before {
		kept++
	}
	s.slices = append(s.slices[:0], s.slices[kept:]...)
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
		skews []float64
		busy  []time.Duration
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
	}

	first := w.firstSliceOf(w.sliceOf(now))
	spentSince := now.Add(-w.config.SweepInterval)
	for payer, s := range w.spenders {
		s.forget(first)
		if len(s.slices) == 0 {
			delete(w.spenders, payer)
			continue
		}
		if !s.lastSeen.Before(spentSince) {
			slices, _ := s.tally(w.config.Stamina.Clicks)
			busy = append(busy, time.Duration(slices)*w.config.Stamina.Slice)
		}
	}

	w.mu.Unlock()

	for _, skew := range skews {
		w.onSkew(skew)
	}
	for _, b := range busy {
		w.onBusy(b)
	}
}
