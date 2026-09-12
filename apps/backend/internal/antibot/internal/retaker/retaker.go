// Package retaker watches for the caller that takes a tile back moments after
// losing it, over and over. Speed is not the signal — two humans fighting over a
// tile are fast, and the player clicking back at a bot is the fastest of all.
// Regularity is: a band no hand holds.
package retaker

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "retaker"

type Config struct {
	// ReactionWindow is how soon after losing a tile a re-take counts as a
	// reaction at all. Keep it well above the delays you expect to measure: a
	// reaction past the edge is not missed, it is censored, and the median of
	// what survives reads faster than the caller really is.
	ReactionWindow time.Duration

	// MinReactions is how many reactions must be in hand before anything is
	// said, so one unlucky exchange is never evidence.
	MinReactions int

	// MaxSpread is the p90-p10 of those reactions. On its own it is Suspect: a
	// caller answering at a steady 1s is holding a band no hand holds, but a
	// steady hand at a slow tempo is not impossible.
	MaxSpread time.Duration

	// MaxMedian turns that band into Certain. A tight band that is also faster
	// than a person can decide is the whole signal, and either bound alone bans
	// real players.
	MaxMedian time.Duration

	// TrackWindow is how far back reactions count.
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

// take is the last click that changed a tile's owner.
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
}

func (w *Watchdog) Name() string { return Name }

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	// A click onto a tile the caller's own country already holds changes
	// nothing and publishes nothing, so it is neither a reaction nor something
	// to react to. Counting it would let a caller spam one tile it owns and
	// frame the next honest player to take it.
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

	// Outside the lock: onReaction feeds a histogram, and the lock is the one
	// every clicker queues on.
	if reacted && w.onReaction != nil {
		w.onReaction(delay)
	}

	return w.verdict(click)
}

// Committed records that the click actually changed the tile, which is what
// makes it something the next caller can react to. A refused click changed no
// tile, and a dropped one changed no tile either.
func (w *Watchdog) Committed(click detect.Click) {
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

	// Nothing prunes a caller while it serves a ban, so judging without this
	// re-bans it on hour-old reactions the moment one lapses.
	c.prune(click.At.Add(-w.config.TrackWindow))

	if len(c.reactions) < w.config.MinReactions {
		return detect.Clear, detect.Evidence{}
	}

	delays := make([]time.Duration, 0, len(c.reactions))
	for _, r := range c.reactions {
		delays = append(delays, r.delay)
	}

	median, spread := detect.Spread(delays)

	if spread > w.config.MaxSpread {
		return detect.Clear, detect.Evidence{}
	}

	verdict := detect.Suspect
	if median <= w.config.MaxMedian {
		verdict = detect.Certain
	}

	return verdict, detect.Evidence{
		Rule: "reflex",
		Fields: []detect.Field{
			{Key: "reactions", Value: len(c.reactions)},
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
	c.reactions = append(c.reactions, reaction{at: click.At, delay: delay})

	c.tiles = append(c.tiles, click.Tile)
	if len(c.tiles) > keptTiles {
		c.tiles = append(c.tiles[:0], c.tiles[len(c.tiles)-keptTiles:]...)
	}
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
