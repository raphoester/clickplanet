package hopper

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "hopper"

type Rules struct {
	Enabled bool

	MinAngle float64
	MinGap   time.Duration
	MaxGap   time.Duration

	MinSteps int
	MinShare float64

	CertainSteps int
	CertainShare float64
}

func (r Rules) Validate() error {
	return errors.Join(
		detect.Positive("minAngle", r.MinAngle),
		detect.AtMost("minAngle", r.MinAngle, "half a turn", 180),
		detect.Positive("minGap", r.MinGap),
		detect.AtLeast("maxGap", r.MaxGap, "minGap", r.MinGap),
		detect.Positive("minSteps", r.MinSteps),
		detect.Share("minShare", r.MinShare),
		detect.AtLeast("certainSteps", r.CertainSteps, "minSteps", r.MinSteps),
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

func New(config Config, rules func() Rules, clock cptime.Clock, onShare func(share float64)) *Watchdog {
	return &Watchdog{
		config:  config.withDefaults(),
		rules:   rules,
		clock:   clock,
		onShare: onShare,
		payers:  make(map[string]*payer),
	}
}

type Watchdog struct {
	config  Config
	rules   func() Rules
	clock   cptime.Clock
	onShare func(float64)

	mu     sync.Mutex
	payers map[string]*payer
}

var _ detect.Watchdog = (*Watchdog)(nil)

type payer struct {
	last   detect.Point
	lastAt time.Time
	seen   bool

	steps []step
}

type step struct {
	at  time.Time
	hop bool
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Enabled() bool { return w.rules().Enabled }

func (w *Watchdog) Attempted(click detect.Click) {
	if !click.Position.Known() {
		return
	}

	rules := w.rules()
	hop := math.Cos(rules.MinAngle * math.Pi / 180)

	w.mu.Lock()
	defer w.mu.Unlock()

	key := click.Payer()
	p, ok := w.payers[key]
	if !ok {
		p = &payer{}
		w.payers[key] = p
	}

	if gap := click.At.Sub(p.lastAt); p.seen && gap >= rules.MinGap && gap <= rules.MaxGap {
		p.steps = append(p.steps, step{at: click.At, hop: cosine(p.last, click.Position) <= hop})
		if kept := capacity(rules); len(p.steps) > kept {
			p.steps = append(p.steps[:0], p.steps[len(p.steps)-kept:]...)
		}
	}

	p.last, p.lastAt, p.seen = click.Position, click.At, true
	p.prune(click.At.Add(-w.config.TrackWindow))
}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	rules := w.rules()

	w.mu.Lock()
	defer w.mu.Unlock()

	p, ok := w.payers[click.Payer()]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	p.prune(click.At.Add(-w.config.TrackWindow))

	steps, hops := p.count()
	share := ratio(hops, steps)

	verdict := detect.Clear
	switch {
	case rules.CertainShare > 0 && steps >= rules.CertainSteps && share >= rules.CertainShare:
		verdict = detect.Certain
	case rules.MinShare > 0 && steps >= rules.MinSteps && share >= rules.MinShare:
		verdict = detect.Suspect
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "hop",
		Fields: []detect.Field{
			{Key: "steps", Value: steps},
			{Key: "hops", Value: hops},
			{Key: "share", Value: share},
		},
	}
}

func (w *Watchdog) Committed(detect.Click) {}

func capacity(rules Rules) int {
	return min(rules.CertainSteps, maxSamples)
}

func cosine(a, b detect.Point) float64 {
	lengths := math.Sqrt((a.X*a.X + a.Y*a.Y + a.Z*a.Z) * (b.X*b.X + b.Y*b.Y + b.Z*b.Z))
	return (a.X*b.X + a.Y*b.Y + a.Z*b.Z) / lengths
}

func (p *payer) prune(cutoff time.Time) {
	kept := p.steps[:0]
	for _, s := range p.steps {
		if s.at.After(cutoff) {
			kept = append(kept, s)
		}
	}
	p.steps = kept
}

func (p *payer) count() (steps, hops int) {
	for _, s := range p.steps {
		if s.hop {
			hops++
		}
	}
	return len(p.steps), hops
}

func ratio(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
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

	var shares []float64

	w.mu.Lock()

	cutoff := now.Add(-w.config.TrackWindow)
	for key, p := range w.payers {
		p.prune(cutoff)

		steps, hops := p.count()
		switch {
		case steps == 0 && now.Sub(p.lastAt) > rules.MaxGap:
			delete(w.payers, key)
		case steps >= rules.MinSteps:
			shares = append(shares, ratio(hops, steps))
		}
	}

	w.mu.Unlock()

	for _, share := range shares {
		w.onShare(share)
	}
}
