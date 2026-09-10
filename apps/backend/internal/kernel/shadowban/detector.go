// Package shadowban spots a caller that clicks back the instant it loses a
// tile, so its clicks can be accepted and dropped rather than refused.
//
// Refusing teaches: a 403 names the check that tripped, a silent no-op names
// nothing. The signal is not speed — a person in a tile war is fast too — but
// regularity: a bot driven by the update stream answers in a tight band. So a
// caller is flagged on a run of reactions whose median is low AND whose spread
// is small, never on either alone.
package shadowban

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

type Config struct {
	// Off still measures, flags, logs and counts — it only stops dropping.
	Enforce bool

	// How soon after losing a tile a re-take counts as a reaction at all.
	ReactionWindow time.Duration

	// Reactions needed inside TrackWindow before anything is decided.
	MinReactions int

	// Both bounds must hold: median at or under MaxMedian, p90-p10 at or under MaxSpread.
	MaxMedian time.Duration
	MaxSpread time.Duration

	// How far back reactions count, and how long a flag lasts once behaviour stops.
	TrackWindow time.Duration
	BanDuration time.Duration

	SweepInterval time.Duration
}

const (
	// Too narrow censors rather than misses: the tail is dropped and the median of what is left reads faster than it is.
	defaultReactionWindow = 5 * time.Second
	defaultMinReactions   = 12
	defaultMaxMedian      = 250 * time.Millisecond
	defaultMaxSpread      = 120 * time.Millisecond
	defaultTrackWindow    = 5 * time.Minute
	defaultBanDuration    = time.Hour
	defaultSweepInterval  = time.Minute

	// Caps the country tally: real players use a handful, and without a bound a
	// client could spend memory by cycling through every valid code.
	maxTrackedCountries = 16
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
	if c.BanDuration <= 0 {
		c.BanDuration = defaultBanDuration
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

// TileOwner tells a click that takes a tile from one that changes nothing, so
// that a caller spamming a tile it already owns is not mistaken for an exchange.
type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

// Report is the evidence behind a flag; the scope it belongs to travels beside
// it, so what reaches a metric and what reaches a log line can differ.
type Report struct {
	Reactions int
	Median    time.Duration
	Spread    time.Duration

	// Self-declared by the client and trivially changed: context for whoever
	// reads the log, never an input to the decision.
	TopCountry       string
	TopCountryClicks int
	Clicks           int

	// A few of the tiles involved, most recent last.
	Tiles []uint32
}

func New(
	config Config,
	owner TileOwner,
	timeProvider xtime.Provider,
	onReaction func(time.Duration),
	onFlag func(scope string, report Report),
) *Detector {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	return &Detector{
		config:       config.withDefaults(),
		owner:        owner,
		timeProvider: timeProvider,
		onReaction:   onReaction,
		onFlag:       onFlag,
		tiles:        make(map[uint32]take),
		callers:      make(map[string]*caller),
	}
}

type Detector struct {
	config       Config
	owner        TileOwner
	timeProvider xtime.Provider

	onReaction func(time.Duration)
	onFlag     func(scope string, report Report)

	mu      sync.Mutex
	tiles   map[uint32]take
	callers map[string]*caller
}

// take is the last click that changed a tile's owner.
type take struct {
	scope string
	at    time.Time
}

type caller struct {
	lastSeen time.Time

	reactions []reaction
	tiles     []uint32

	clicks    int
	countries map[string]int

	bannedUntil time.Time
}

type reaction struct {
	at    time.Time
	delay time.Duration
}

// Observe judges a click before it is handled. It returns whether to drop it,
// and whether the caller should report back through Took once the handler has
// accepted it — a click that is dropped, refused or a no-op takes nothing, and
// recording one anyway is how a griefer gets an honest player flagged.
func (d *Detector) Observe(scope string, tile uint32, country string) (drop bool, takes bool) {
	if scope == "" {
		return false, false
	}

	now := d.timeProvider.Now()

	d.mu.Lock()
	c := d.callerLocked(scope, now)
	c.lastSeen = now
	c.clicks++
	c.countCountryLocked(country)

	// A click onto a tile that country already holds changes nothing, so it
	// neither reacts to the previous holder nor becomes an event to react to.
	if held, ok := d.owner.Owner(tile); ok && held == country {
		banned := d.bannedLocked(c, now)
		d.mu.Unlock()
		return banned, false
	}

	var (
		delay   time.Duration
		reacted bool
	)

	if previous, ok := d.tiles[tile]; ok && previous.scope != scope {
		if delay = now.Sub(previous.at); delay >= 0 && delay <= d.config.ReactionWindow {
			reacted = true
			c.addReactionLocked(now, delay, tile, d.config)
		}
	}

	report, flagged := d.evaluateLocked(c, now)
	banned := d.bannedLocked(c, now)
	d.mu.Unlock()

	// Outside the lock: onFlag writes a log line, and holding the map through
	// that would queue every other clicker behind the I/O.
	if reacted && d.onReaction != nil {
		d.onReaction(delay)
	}
	if flagged && d.onFlag != nil {
		d.onFlag(scope, report)
	}

	return banned, !banned
}

// Took records that a click Observe cleared went on to change the tile. It is
// what a later click can be a reaction to.
func (d *Detector) Took(scope string, tile uint32) {
	if scope == "" {
		return
	}

	now := d.timeProvider.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.tiles[tile] = take{scope: scope, at: now}
}

// Flagged counts callers inside a ban, enforced or not — what the gauge reports.
func (d *Detector) Flagged() int {
	now := d.timeProvider.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	var count int
	for _, c := range d.callers {
		if now.Before(c.bannedUntil) {
			count++
		}
	}

	return count
}

func (d *Detector) callerLocked(scope string, now time.Time) *caller {
	c, ok := d.callers[scope]
	if !ok {
		c = &caller{lastSeen: now, countries: make(map[string]int)}
		d.callers[scope] = c
	}
	return c
}

func (d *Detector) bannedLocked(c *caller, now time.Time) bool {
	return d.config.Enforce && now.Before(c.bannedUntil)
}

// evaluateLocked reports a flag once per ban, not once per click inside one.
func (d *Detector) evaluateLocked(c *caller, now time.Time) (Report, bool) {
	if now.Before(c.bannedUntil) {
		return Report{}, false
	}

	// Nothing prunes a caller while it serves a ban, so judging without this re-bans it on hour-old reactions the moment one lapses.
	c.pruneReactionsLocked(now.Add(-d.config.TrackWindow))

	if len(c.reactions) < d.config.MinReactions {
		return Report{}, false
	}

	delays := make([]time.Duration, 0, len(c.reactions))
	for _, r := range c.reactions {
		delays = append(delays, r.delay)
	}
	sort.Slice(delays, func(i, j int) bool { return delays[i] < delays[j] })

	median := quantile(delays, 0.5)
	spread := quantile(delays, 0.9) - quantile(delays, 0.1)

	if median > d.config.MaxMedian || spread > d.config.MaxSpread {
		return Report{}, false
	}

	c.bannedUntil = now.Add(d.config.BanDuration)

	country, countryClicks := c.topCountryLocked()

	return Report{
		Reactions:        len(c.reactions),
		Median:           median,
		Spread:           spread,
		TopCountry:       country,
		TopCountryClicks: countryClicks,
		Clicks:           c.clicks,
		Tiles:            append([]uint32(nil), c.tiles...),
	}, true
}

func (c *caller) pruneReactionsLocked(cutoff time.Time) {
	kept := c.reactions[:0]
	for _, r := range c.reactions {
		if r.at.After(cutoff) {
			kept = append(kept, r)
		}
	}
	c.reactions = kept
}

func (c *caller) addReactionLocked(now time.Time, delay time.Duration, tile uint32, config Config) {
	c.pruneReactionsLocked(now.Add(-config.TrackWindow))
	c.reactions = append(c.reactions, reaction{at: now, delay: delay})

	const keptTiles = 8
	c.tiles = append(c.tiles, tile)
	if len(c.tiles) > keptTiles {
		c.tiles = append(c.tiles[:0], c.tiles[len(c.tiles)-keptTiles:]...)
	}
}

func (c *caller) countCountryLocked(country string) {
	if country == "" {
		return
	}
	if _, known := c.countries[country]; !known && len(c.countries) >= maxTrackedCountries {
		return
	}
	c.countries[country]++
}

func (c *caller) topCountryLocked() (string, int) {
	var (
		top   string
		count int
	)

	for country, n := range c.countries {
		// Ties break on the code, so the same tally always names the same country.
		if n > count || (n == count && country < top) {
			top, count = country, n
		}
	}

	return top, count
}

func (d *Detector) Run(ctx context.Context) {
	ticker := time.NewTicker(d.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.sweep()
		case <-ctx.Done():
			return
		}
	}
}

// sweep forgets what can no longer affect a decision; without it both maps keep
// an entry per tile and per caller that ever clicked.
func (d *Detector) sweep() {
	now := d.timeProvider.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	tileCutoff := now.Add(-d.config.ReactionWindow)
	for tile, t := range d.tiles {
		if t.at.Before(tileCutoff) {
			delete(d.tiles, tile)
		}
	}

	callerCutoff := now.Add(-d.config.TrackWindow)
	for scope, c := range d.callers {
		if now.Before(c.bannedUntil) {
			continue
		}
		if c.lastSeen.Before(callerCutoff) {
			delete(d.callers, scope)
			continue
		}

		c.pruneReactionsLocked(callerCutoff)
	}
}

// quantile rounds to the nearest sample rather than interpolating, so every
// number in the log line is a delay that was actually measured.
func quantile(sorted []time.Duration, q float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	i := int(q * float64(len(sorted)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}

	return sorted[i]
}
