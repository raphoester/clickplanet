package retaker

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "retaker"

type Config struct {
	ReactionWindow time.Duration

	MinReactions int

	MaxSpread time.Duration

	MaxMedian time.Duration

	MinTiles   int
	RoamMedian time.Duration

	CertainTiles int

	TrackWindow time.Duration

	SweepInterval time.Duration
}

const (
	defaultReactionWindow = 5 * time.Second
	defaultMinReactions   = 12
	defaultMaxMedian      = 250 * time.Millisecond
	defaultMaxSpread      = 120 * time.Millisecond
	defaultTrackWindow    = 5 * time.Minute
	defaultSweepInterval  = time.Minute

	keptTiles = 8
)

func (c Config) withDefaults() Config {
	if c.ReactionWindow <= 0 {
		c.ReactionWindow = defaultReactionWindow
	}
	if c.MinReactions <= 0 {
		c.MinReactions = defaultMinReactions
	}
	if c.MaxMedian <= 0 {
		c.MaxMedian = defaultMaxMedian
	}
	if c.MaxSpread <= 0 {
		c.MaxSpread = defaultMaxSpread
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

func New(config Config, clock cptime.Clock, onReaction func(time.Duration)) *Watchdog {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Watchdog{
		config:     config.withDefaults(),
		clock:      clock,
		onReaction: onReaction,
		tiles:      make(map[uint32]take),
		callers:    make(map[string]*caller),
	}
}

type Watchdog struct {
	config     Config
	clock      cptime.Clock
	onReaction func(time.Duration)

	mu      sync.Mutex
	tiles   map[uint32]take
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

type take struct {
	scope string
	at    time.Time
}

type caller struct {
	reactions []reaction
	tiles     []uint32
}

type reaction struct {
	at    time.Time
	delay time.Duration
	tile  uint32
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Attempted(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	if click.NoOp {
		return w.verdict(click)
	}

	var (
		delay   time.Duration
		reacted bool
	)

	w.mu.Lock()
	if previous, ok := w.tiles[click.Tile]; ok && previous.scope != click.Scope {
		if delay = click.At.Sub(previous.at); delay >= 0 && delay <= w.config.ReactionWindow {
			reacted = true
			w.callerLocked(click.Scope).addReaction(click, delay, w.config.TrackWindow)
		}
	}
	w.mu.Unlock()

	if reacted && w.onReaction != nil {
		w.onReaction(delay)
	}

	return w.verdict(click)
}

func (w *Watchdog) Committed(click detect.Click) {
	// A no-op is not a take: recording it would let a caller frame the next honest clicker.
	if click.NoOp || click.Scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.tiles[click.Tile] = take{scope: click.Scope, at: click.At}
}

func (w *Watchdog) verdict(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	c.prune(click.At.Add(-w.config.TrackWindow))

	if len(c.reactions) < w.config.MinReactions {
		return detect.Clear, detect.Evidence{}
	}

	delays := make([]time.Duration, 0, len(c.reactions))
	for _, r := range c.reactions {
		delays = append(delays, r.delay)
	}

	median, spread := detect.Spread(delays)

	band := detect.Clear
	if spread <= w.config.MaxSpread {
		band = detect.Suspect
		if median <= w.config.MaxMedian {
			band = detect.Certain
		}
	}

	tiles := c.distinctTiles()
	roam := detect.Clear
	if w.config.MinTiles > 0 && tiles >= w.config.MinTiles && median <= w.config.RoamMedian {
		roam = detect.Suspect
		if w.config.CertainTiles > 0 && tiles >= w.config.CertainTiles {
			roam = detect.Certain
		}
	}

	if band == detect.Clear && roam == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	verdict, rule := band, "reflex"
	if roam > band {
		verdict, rule = roam, "roam"
	}

	return verdict, detect.Evidence{
		Rule: rule,
		Fields: []detect.Field{
			{Key: "reactions", Value: len(c.reactions)},
			{Key: "tiles", Value: tiles},
			{Key: "median", Value: median},
			{Key: "spread", Value: spread},
			{Key: "reactedOn", Value: append([]uint32(nil), c.tiles...)},
		},
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

func (c *caller) addReaction(click detect.Click, delay time.Duration, window time.Duration) {
	c.prune(click.At.Add(-window))
	c.reactions = append(c.reactions, reaction{at: click.At, delay: delay, tile: click.Tile})

	c.tiles = append(c.tiles, click.Tile)
	if len(c.tiles) > keptTiles {
		c.tiles = append(c.tiles[:0], c.tiles[len(c.tiles)-keptTiles:]...)
	}
}

func (c *caller) distinctTiles() int {
	seen := cpcolls.NewSetWithCapacity[uint32](len(c.reactions))
	for _, r := range c.reactions {
		seen.Add(r.tile)
	}
	return seen.Len()
}

func (c *caller) prune(cutoff time.Time) {
	kept := c.reactions[:0]
	for _, r := range c.reactions {
		if r.at.After(cutoff) {
			kept = append(kept, r)
		}
	}
	c.reactions = kept
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

	tileCutoff := now.Add(-w.config.ReactionWindow)
	for tile, t := range w.tiles {
		if t.at.Before(tileCutoff) {
			delete(w.tiles, tile)
		}
	}

	callerCutoff := now.Add(-w.config.TrackWindow)
	for scope, c := range w.callers {
		c.prune(callerCutoff)
		if len(c.reactions) == 0 {
			delete(w.callers, scope)
		}
	}
}
