package hopper

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "hopper"

type Config struct {
	MinAngle float64
	MinGap   time.Duration
	MaxGap   time.Duration

	MinSteps int
	MinShare float64

	CertainSteps int
	CertainShare float64

	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultMinAngle      = 45
	defaultMinGap        = 100 * time.Millisecond
	defaultMaxGap        = 30 * time.Second
	defaultMinSteps      = 40
	defaultCertainSteps  = 200
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	maxSamples = 4096
)

func (c Config) withDefaults() Config {
	if c.MinAngle <= 0 {
		c.MinAngle = defaultMinAngle
	}
	if c.MinGap <= 0 {
		c.MinGap = defaultMinGap
	}
	if c.MaxGap <= 0 {
		c.MaxGap = defaultMaxGap
	}
	if c.MinSteps <= 0 {
		c.MinSteps = defaultMinSteps
	}
	if c.CertainSteps <= 0 {
		c.CertainSteps = defaultCertainSteps
	}
	if c.CertainSteps < c.MinSteps {
		c.CertainSteps = c.MinSteps
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock, onShare func(share float64)) *Watchdog {
	config = config.withDefaults()

	return &Watchdog{
		config:  config,
		clock:   clock,
		onShare: onShare,
		hop:     math.Cos(config.MinAngle * math.Pi / 180),
		payers:  make(map[string]*payer),
	}
}

type Watchdog struct {
	config  Config
	clock   cptime.Clock
	onShare func(float64)
	hop     float64

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

func (w *Watchdog) Attempted(click detect.Click) {
	if !click.Position.Known() {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	key := click.Payer()
	p, ok := w.payers[key]
	if !ok {
		p = &payer{}
		w.payers[key] = p
	}

	if gap := click.At.Sub(p.lastAt); p.seen && gap >= w.config.MinGap && gap <= w.config.MaxGap {
		p.steps = append(p.steps, step{at: click.At, hop: cosine(p.last, click.Position) <= w.hop})
		if capacity := w.capacity(); len(p.steps) > capacity {
			p.steps = append(p.steps[:0], p.steps[len(p.steps)-capacity:]...)
		}
	}

	p.last, p.lastAt, p.seen = click.Position, click.At, true
	p.prune(click.At.Add(-w.config.TrackWindow))
}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
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
	case w.config.CertainShare > 0 && steps >= w.config.CertainSteps && share >= w.config.CertainShare:
		verdict = detect.Certain
	case w.config.MinShare > 0 && steps >= w.config.MinSteps && share >= w.config.MinShare:
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

func (w *Watchdog) capacity() int {
	if w.config.CertainSteps > maxSamples {
		return maxSamples
	}
	return w.config.CertainSteps
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

	var shares []float64

	w.mu.Lock()

	cutoff := now.Add(-w.config.TrackWindow)
	for key, p := range w.payers {
		p.prune(cutoff)

		steps, hops := p.count()
		switch {
		case steps == 0 && now.Sub(p.lastAt) > w.config.MaxGap:
			delete(w.payers, key)
		case steps >= w.config.MinSteps:
			shares = append(shares, ratio(hops, steps))
		}
	}

	w.mu.Unlock()

	for _, share := range shares {
		w.onShare(share)
	}
}
