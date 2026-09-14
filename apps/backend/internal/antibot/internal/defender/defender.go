// Package defender watches for the caller whose clicks are nearly all retakes: a
// tile its country lost a moment ago, taken back. The retaker times how fast that
// happens, and a bot working a queue behind the throttle is not fast — the tenth
// tile it takes back waits ten refills. What it cannot hide is what it clicks.
// A person paints new ground between fights; a defence loop does nothing else.
package defender

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "defender"

type Config struct {
	// RetakeWindow is how long after a country loses a tile a take for that
	// country counts as a retake. A queue drains at the refill rate, so this is
	// sized to the queue and not to a reflex.
	RetakeWindow time.Duration

	// MinClicks and MinShare read Suspect: that many takes in TrackWindow, and
	// that share of them retakes. A zero share never reads Suspect.
	MinClicks int
	MinShare  float64

	// CertainClicks and CertainShare read Certain. A zero share never reads Certain.
	CertainClicks int
	CertainShare  float64

	// TrackWindow is how far back takes count.
	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultRetakeWindow  = 2 * time.Minute
	defaultMinClicks     = 60
	defaultCertainClicks = 200
	defaultTrackWindow   = 5 * time.Minute
	defaultSweepInterval = time.Minute

	maxSamples = 1024
)

// withDefaults leaves both shares alone: unset, the watchdog measures and says nothing.
func (c Config) withDefaults() Config {
	if c.RetakeWindow <= 0 {
		c.RetakeWindow = defaultRetakeWindow
	}
	if c.MinClicks <= 0 {
		c.MinClicks = defaultMinClicks
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

// New takes onShare, called each sweep with the share of every caller past MinClicks.
func New(config Config, clock cptime.Clock, onShare func(share float64)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:  config.withDefaults(),
		clock:   clock,
		onShare: onShare,
		losses:  make(map[uint32]loss),
		callers: make(map[string]*caller),
	}
}

type Watchdog struct {
	config  Config
	clock   cptime.Clock
	onShare func(float64)

	mu      sync.Mutex
	losses  map[uint32]loss
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

// loss is the last take that changed a tile's owner: who lost it, to whom, when.
type loss struct {
	country string
	to      string
	at      time.Time
}

type caller struct {
	takes []take
}

type take struct {
	at     time.Time
	retake bool
}

func (w *Watchdog) Name() string { return Name }

// Attempted is nothing to this watchdog. A refused click took nothing back.
func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// A click onto a tile the caller's country holds is neither a take nor a retake.
	if !click.NoOp {
		previous, ok := w.losses[click.Tile]
		retake := ok &&
			previous.country == click.Country &&
			previous.to != click.Scope &&
			click.At.Sub(previous.at) <= w.config.RetakeWindow

		c := w.callerLocked(click.Scope)
		c.takes = append(c.takes, take{at: click.At, retake: retake})
		if len(c.takes) > maxSamples {
			c.takes = append(c.takes[:0], c.takes[len(c.takes)-maxSamples:]...)
		}
	}

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	c.prune(click.At.Add(-w.config.TrackWindow))

	clicks, retakes := c.count()

	verdict := detect.Clear
	share := ratio(retakes, clicks)

	switch {
	case w.config.CertainShare > 0 && clicks >= w.config.CertainClicks && share >= w.config.CertainShare:
		verdict = detect.Certain
	case w.config.MinShare > 0 && clicks >= w.config.MinClicks && share >= w.config.MinShare:
		verdict = detect.Suspect
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: "defence",
		Fields: []detect.Field{
			{Key: "clicks", Value: clicks},
			{Key: "retakes", Value: retakes},
			{Key: "share", Value: share},
		},
	}
}

// Committed records who lost the tile. Only a click that reached the map took it from anyone.
func (w *Watchdog) Committed(click detect.Click) {
	if click.NoOp || click.Held == "" || click.Scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.losses[click.Tile] = loss{country: click.Held, to: click.Scope, at: click.At}
}

func (w *Watchdog) callerLocked(scope string) *caller {
	c, ok := w.callers[scope]
	if !ok {
		c = &caller{}
		w.callers[scope] = c
	}
	return c
}

func (c *caller) prune(cutoff time.Time) {
	kept := c.takes[:0]
	for _, t := range c.takes {
		if t.at.After(cutoff) {
			kept = append(kept, t)
		}
	}
	c.takes = kept
}

func (c *caller) count() (clicks, retakes int) {
	for _, t := range c.takes {
		if t.retake {
			retakes++
		}
	}
	return len(c.takes), retakes
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

// sweep forgets what can no longer matter and reports the shares still in hand.
func (w *Watchdog) sweep() {
	now := w.clock.Now()

	var shares []float64

	w.mu.Lock()

	lossCutoff := now.Add(-w.config.RetakeWindow)
	for tile, l := range w.losses {
		if l.at.Before(lossCutoff) {
			delete(w.losses, tile)
		}
	}

	callerCutoff := now.Add(-w.config.TrackWindow)
	for scope, c := range w.callers {
		c.prune(callerCutoff)

		clicks, retakes := c.count()
		switch {
		case clicks == 0:
			delete(w.callers, scope)
		case clicks >= w.config.MinClicks:
			shares = append(shares, ratio(retakes, clicks))
		}
	}

	w.mu.Unlock()

	// Outside the lock: it feeds a histogram, and the lock is the one every clicker queues on.
	if w.onShare != nil {
		for _, share := range shares {
			w.onShare(share)
		}
	}
}
