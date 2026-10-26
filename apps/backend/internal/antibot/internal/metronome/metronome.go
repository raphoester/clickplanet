package metronome

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "metronome"

type Rules struct {
	Enabled bool

	MaxGap time.Duration

	MaxSpread time.Duration

	MinClicks int

	CertainFor    time.Duration
	CertainClicks int

	Shape ShapeRules

	Clock ClockRules

	Stamina StaminaRules
}

type ShapeRules struct {
	MaxGap time.Duration

	// Pointers because 0 is a skew: nil never reads its level.
	Clicks        int
	MaxSkew       *float64
	CertainClicks int
	CertainSkew   *float64
}

type ClockRules struct {
	Period time.Duration

	Clicks           int
	MinCoherence     *float64
	CertainClicks    int
	CertainFor       time.Duration
	CertainCoherence *float64
}

type StaminaRules struct {
	Clicks      int
	Window      time.Duration
	MinBusy     time.Duration
	CertainBusy time.Duration
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("maxGap", r.MaxGap),
		detect.Positive("maxSpread", r.MaxSpread),
		detect.Positive("minClicks", r.MinClicks),
		detect.Positive("certainFor", r.CertainFor),
		detect.AtLeast("certainClicks", r.CertainClicks, "minClicks", r.MinClicks),
		r.Shape.validate(),
		r.Clock.validate(),
		r.Stamina.validate(),
	)
}

func (r ShapeRules) validate() error {
	return errors.Join(
		detect.Positive("shape.maxGap", r.MaxGap),
		detect.Positive("shape.clicks", r.Clicks),
		detect.AtMost("shape.clicks", r.Clicks, "the gaps kept", maxShapeSamples),
		detect.AtLeast("shape.certainClicks", r.CertainClicks, "shape.clicks", r.Clicks),
		detect.AtMost("shape.certainClicks", r.CertainClicks, "the gaps kept", maxShapeSamples),
		skew("shape.maxSkew", r.MaxSkew),
		skew("shape.certainSkew", r.CertainSkew),
	)
}

func skew(name string, value *float64) error {
	if value == nil {
		return nil
	}
	return errors.Join(
		detect.AtLeast(name, *value, "the lowest skew", -1),
		detect.AtMost(name, *value, "the highest skew", 1),
	)
}

func (r ClockRules) validate() error {
	return errors.Join(
		detect.Positive("clock.period", r.Period),
		detect.Positive("clock.clicks", r.Clicks),
		detect.AtMost("clock.clicks", r.Clicks, "the tries kept", maxClockSamples),
		detect.AtLeast("clock.certainClicks", r.CertainClicks, "clock.clicks", r.Clicks),
		detect.AtMost("clock.certainClicks", r.CertainClicks, "the tries kept", maxClockSamples),
		detect.Positive("clock.certainFor", r.CertainFor),
		coherence("clock.minCoherence", r.MinCoherence),
		coherence("clock.certainCoherence", r.CertainCoherence),
	)
}

func coherence(name string, value *float64) error {
	if value == nil {
		return nil
	}
	return detect.Share(name, *value) //nolint:wrapcheck // it names the bound it refuses.
}

func (r StaminaRules) validate() error {
	return errors.Join(
		detect.Positive("stamina.clicks", r.Clicks),
		detect.Positive("stamina.window", r.Window),
		detect.NotNegative("stamina.minBusy", r.MinBusy),
		detect.NotNegative("stamina.certainBusy", r.CertainBusy),
	)
}

type Config struct {
	Stamina StaminaConfig

	TrackWindow time.Duration

	SweepInterval time.Duration
}

type StaminaConfig struct {
	// The clicks are counted per slice, and saved with its length.
	Slice time.Duration
}

const (
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	defaultStaminaSlice = 10 * time.Minute

	maxSamples      = 1024
	maxShapeSamples = 2048
	maxClockSamples = 2048
)

func (c Config) withDefaults() Config {
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	if c.Stamina.Slice <= 0 {
		c.Stamina.Slice = defaultStaminaSlice
	}
	return c
}

func (r ClockRules) kept() int {
	switch {
	case r.CertainCoherence != nil:
		return r.CertainClicks
	case r.MinCoherence != nil:
		return r.Clicks
	default:
		return 0
	}
}

func New(
	config Config,
	rules func() Rules,
	clock cptime.Clock,
	onSkew, onCoherence func(float64),
	onBusy func(busy time.Duration),
) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:      config.withDefaults(),
		rules:       rules,
		clock:       clock,
		onSkew:      onSkew,
		onCoherence: onCoherence,
		onBusy:      onBusy,
		payers:      make(map[string]*payer),
		spenders:    make(map[string]*spender),
	}
}

type Watchdog struct {
	config      Config
	rules       func() Rules
	clock       cptime.Clock
	onSkew      func(float64)
	onCoherence func(float64)
	onBusy      func(time.Duration)

	mu       sync.Mutex
	payers   map[string]*payer
	spenders map[string]*spender
	outage   detect.Outage
}

var _ detect.Watchdog = (*Watchdog)(nil)

type payer struct {
	lastSeen time.Time

	runStart  time.Time
	runClicks int

	gaps []time.Duration

	// shape and tries survive a break: the bots they catch pause between bursts.
	shape []time.Duration

	tries []time.Time
}

type spender struct {
	lastSeen time.Time
	slices   []slice
}

type slice struct {
	index  int64
	clicks int
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Enabled() bool { return w.rules().Enabled }

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Attempted(click detect.Click) {
	rules := w.rules()

	w.mu.Lock()
	defer w.mu.Unlock()

	key := click.Payer()
	p, ok := w.payers[key]
	if !ok {
		p = &payer{}
		w.payers[key] = p
		p.restart(click.At)
		p.tried(click.At, rules.Clock.kept())
		return
	}

	p.tried(click.At, rules.Clock.kept())

	across := w.outage.Across(p.lastSeen, click.At)
	gap := w.outage.Gap(p.lastSeen, click.At)
	p.lastSeen = click.At

	if !across && gap >= 0 && gap <= rules.Shape.MaxGap {
		p.shape = append(p.shape, gap)
		if kept := rules.Shape.CertainClicks; len(p.shape) > kept {
			p.shape = append(p.shape[:0], p.shape[len(p.shape)-kept:]...)
		}
	}

	if gap < 0 || gap > rules.MaxGap {
		p.restart(click.At)
		return
	}

	p.runClicks++

	if across {
		p.runStart = p.runStart.Add(w.outage.Length())
		return
	}

	p.gaps = append(p.gaps, gap)
	if kept := capacity(rules); len(p.gaps) > kept {
		p.gaps = append(p.gaps[:0], p.gaps[len(p.gaps)-kept:]...)
	}
}

func (p *payer) tried(at time.Time, kept int) {
	if kept == 0 {
		return
	}

	p.tries = append(p.tries, at)
	if len(p.tries) > kept {
		p.tries = append(p.tries[:0], p.tries[len(p.tries)-kept:]...)
	}
}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	rules := w.rules()

	w.mu.Lock()
	defer w.mu.Unlock()

	key := click.Payer()
	w.spend(key, click.At, rules.Stamina.Window)
	stamina, staminaEvidence := w.stamina(w.spenders[key], rules.Stamina)

	p, ok := w.payers[key]
	if !ok {
		return stamina, staminaEvidence
	}

	verdict, evidence := cadence(p, rules)
	if shape, shapeEvidence := shape(p, rules.Shape); shape > verdict {
		verdict, evidence = shape, shapeEvidence
	}
	if timer, timerEvidence := timer(p, rules.Clock); timer > verdict {
		verdict, evidence = timer, timerEvidence
	}
	if stamina > verdict {
		verdict, evidence = stamina, staminaEvidence
	}
	return verdict, evidence
}

func (w *Watchdog) spend(key string, at time.Time, window time.Duration) {
	s, ok := w.spenders[key]
	if !ok {
		s = &spender{}
		w.spenders[key] = s
	}
	s.lastSeen = at

	index := w.sliceOf(at)
	if last := len(s.slices) - 1; last >= 0 && index <= s.slices[last].index {
		s.slices[last].clicks++
	} else {
		s.slices = append(s.slices, slice{index: index, clicks: 1})
	}

	s.forget(w.firstSliceOf(index, window))
}

func (w *Watchdog) stamina(s *spender, rules StaminaRules) (detect.Verdict, detect.Evidence) {
	busySlices, clicks := s.tally(rules.Clicks)
	busy := time.Duration(busySlices) * w.config.Stamina.Slice

	var verdict detect.Verdict
	switch {
	case rules.CertainBusy > 0 && busy >= rules.CertainBusy:
		verdict = detect.Certain
	case rules.MinBusy > 0 && busy >= rules.MinBusy:
		verdict = detect.Suspect
	default:
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "stamina",
		Zeal: true,
		Fields: []detect.Field{
			{Key: "busy", Value: busy},
			{Key: "window", Value: rules.Window},
			{Key: "clicks", Value: clicks},
		},
	}
}

func (w *Watchdog) sliceOf(at time.Time) int64 {
	return at.UnixNano() / int64(w.config.Stamina.Slice)
}

func (w *Watchdog) firstSliceOf(index int64, window time.Duration) int64 {
	return index - max(int64(window/w.config.Stamina.Slice), 1) + 1
}

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

func cadence(p *payer, rules Rules) (detect.Verdict, detect.Evidence) {
	if p.runClicks < rules.MinClicks {
		return detect.Clear, detect.Evidence{}
	}

	median, spread := detect.Spread(append([]time.Duration(nil), p.gaps...))
	if spread > rules.MaxSpread {
		return detect.Clear, detect.Evidence{}
	}

	sustained := p.lastSeen.Sub(p.runStart)

	verdict := detect.Suspect
	if sustained >= rules.CertainFor && p.runClicks >= rules.CertainClicks {
		verdict = detect.Certain
	}

	return verdict, detect.Evidence{
		Rule: "cadence",
		Fields: []detect.Field{
			{Key: "spread", Value: spread},
			{Key: "median", Value: median},
			{Key: "clicks", Value: p.runClicks},
			{Key: "sustained", Value: sustained},
		},
	}
}

func shape(p *payer, rules ShapeRules) (detect.Verdict, detect.Evidence) {
	if rules.CertainSkew != nil {
		if s, ok := skewOf(p.shape, rules.CertainClicks); ok && s.skew <= *rules.CertainSkew {
			return detect.Certain, s.evidence()
		}
	}

	if rules.MaxSkew != nil {
		if s, ok := skewOf(p.shape, rules.Clicks); ok && s.skew <= *rules.MaxSkew {
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

func timer(p *payer, rules ClockRules) (detect.Verdict, detect.Evidence) {
	if rules.CertainCoherence != nil {
		if b, ok := beatOf(p.tries, rules.CertainClicks, rules.Period); ok &&
			b.coherence >= *rules.CertainCoherence && b.span >= rules.CertainFor {
			return detect.Certain, b.evidence()
		}
	}

	if rules.MinCoherence != nil {
		if b, ok := beatOf(p.tries, rules.Clicks, rules.Period); ok && b.coherence >= *rules.MinCoherence {
			return detect.Suspect, b.evidence()
		}
	}

	return detect.Clear, detect.Evidence{}
}

type beat struct {
	coherence float64
	period    time.Duration
	offset    time.Duration
	span      time.Duration
	clicks    int
}

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

func capacity(rules Rules) int {
	return min(rules.MinClicks, maxSamples)
}

func (p *payer) restart(at time.Time) {
	p.lastSeen = at
	p.runStart = at
	p.runClicks = 1
	p.gaps = p.gaps[:0]
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
	rules := w.rules()

	var (
		skews      []float64
		coherences []float64
		busy       []time.Duration
	)

	w.mu.Lock()

	cutoff := now.Add(-w.config.TrackWindow)
	for key, p := range w.payers {
		if p.lastSeen.Before(cutoff) {
			delete(w.payers, key)
			continue
		}
		if s, ok := skewOf(p.shape, rules.Shape.Clicks); ok {
			skews = append(skews, s.skew)
		}
		if b, ok := beatOf(p.tries, rules.Clock.Clicks, rules.Clock.Period); ok {
			coherences = append(coherences, b.coherence)
		}
	}

	first := w.firstSliceOf(w.sliceOf(now), rules.Stamina.Window)
	spentSince := now.Add(-w.config.SweepInterval)
	for key, s := range w.spenders {
		s.forget(first)
		if len(s.slices) == 0 {
			delete(w.spenders, key)
			continue
		}
		if !s.lastSeen.Before(spentSince) {
			slices, _ := s.tally(rules.Stamina.Clicks)
			busy = append(busy, time.Duration(slices)*w.config.Stamina.Slice)
		}
	}

	w.mu.Unlock()

	for _, skew := range skews {
		w.onSkew(skew)
	}
	for _, coherence := range coherences {
		w.onCoherence(coherence)
	}
	for _, b := range busy {
		w.onBusy(b)
	}
}
