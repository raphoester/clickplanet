// Package scraper watches for the caller that reads the whole map again and again, beyond one read
// per stream opened, and for the one that reads it off the lattice the web app walks.
package scraper

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const Name = "scraper"

type Config struct {
	// MinMaps is the whole maps read beyond one per stream opened, inside TrackWindow, that read as Suspect.
	MinMaps float64

	// CertainMaps is the same count at a pace no page load produces.
	CertainMaps float64

	// CertainOffMap is the reads off the map, inside TrackWindow, that read Certain however little was read.
	CertainOffMap int

	TrackWindow   time.Duration
	SweepInterval time.Duration
}

const (
	defaultMinMaps       = 5
	defaultCertainMaps   = 15
	defaultCertainOffMap = 2
	defaultTrackWindow   = 15 * time.Minute
	defaultSweepInterval = time.Minute

	// steps bounds what one caller costs, however fast it reads: GetMap is not throttled.
	steps = 30
)

func (c Config) withDefaults() Config {
	if c.MinMaps <= 0 {
		c.MinMaps = defaultMinMaps
	}
	if c.CertainMaps <= 0 {
		c.CertainMaps = defaultCertainMaps
	}
	if c.CertainMaps < c.MinMaps {
		c.CertainMaps = c.MinMaps
	}
	if c.CertainOffMap <= 0 {
		c.CertainOffMap = defaultCertainOffMap
	}
	if c.TrackWindow <= 0 {
		c.TrackWindow = defaultTrackWindow
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

// New takes onMaps, called each sweep with the unexplained maps of every caller that clicked inside TrackWindow.
func New(config Config, clock cptime.Clock, onMaps func(maps float64)) *Watchdog {
	config = config.withDefaults()

	return &Watchdog{
		config:  config,
		step:    config.TrackWindow / steps,
		clock:   clock,
		onMaps:  onMaps,
		callers: make(map[string]*caller),
	}
}

type Watchdog struct {
	config Config
	step   time.Duration
	clock  cptime.Clock
	onMaps func(float64)

	mu      sync.Mutex
	callers map[string]*caller
}

var _ detect.Watchdog = (*Watchdog)(nil)

type caller struct {
	slices  []slice // oldest first
	clicked time.Time
}

type slice struct {
	at      time.Time
	maps    float64
	streams int
	offMap  int
}

func (w *Watchdog) Name() string { return Name }

// Fetched records a read of maps whole maps: a GetMap of a tenth of the map is 0.1.
// offMap says the read asked for tiles the map does not have — see beyond.
func (w *Watchdog) Fetched(scope string, maps float64, offMap bool) {
	if scope == "" || (maps <= 0 && !offMap) {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	s := w.sliceLocked(scope)
	if maps > 0 {
		s.maps += maps
	}
	if offMap {
		s.offMap++
	}
}

// Listened records a live stream opened, which explains one map read.
func (w *Watchdog) Listened(scope string) {
	if scope == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.sliceLocked(scope).streams++
}

func (w *Watchdog) Attempted(click detect.Click) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if c, ok := w.callers[click.Scope]; ok {
		c.clicked = click.At
	}
}

func (w *Watchdog) Committed(detect.Click) {}

func (w *Watchdog) Watch(click detect.Click) (detect.Verdict, detect.Evidence) {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.callers[click.Scope]
	if !ok {
		return detect.Clear, detect.Evidence{}
	}

	maps, streams, offMap := c.count(click.At.Add(-w.config.TrackWindow))
	unexplained := beyond(maps, streams, offMap)

	verdict, rule := detect.Clear, "poll"
	switch {
	case offMap >= w.config.CertainOffMap:
		verdict, rule = detect.Certain, "offMap"
	case unexplained >= w.config.CertainMaps:
		verdict = detect.Certain
	case unexplained >= w.config.MinMaps:
		verdict = detect.Suspect
	}

	if verdict == detect.Clear {
		return detect.Clear, detect.Evidence{}
	}

	return verdict, detect.Evidence{
		Rule: rule,
		Fields: []detect.Field{
			{Key: "maps", Value: math.Round(maps*10) / 10},
			{Key: "streams", Value: streams},
			{Key: "offMap", Value: offMap},
		},
	}
}

func (w *Watchdog) sliceLocked(scope string) *slice {
	now := w.clock.Now()

	c, ok := w.callers[scope]
	if !ok {
		c = &caller{}
		w.callers[scope] = c
	}

	c.prune(now.Add(-w.config.TrackWindow))

	if n := len(c.slices); n > 0 && now.Sub(c.slices[n-1].at) < w.step {
		return &c.slices[n-1]
	}

	c.slices = append(c.slices, slice{at: now})
	return &c.slices[len(c.slices)-1]
}

// prune drops a slice straddling the cutoff whole, which only ever undercounts.
func (c *caller) prune(cutoff time.Time) {
	kept := c.slices[:0]
	for _, s := range c.slices {
		if !s.at.Before(cutoff) {
			kept = append(kept, s)
		}
	}
	c.slices = kept
}

func (c *caller) count(cutoff time.Time) (maps float64, streams, offMap int) {
	var read float64
	for _, s := range c.slices {
		if s.at.Before(cutoff) {
			continue
		}
		read += s.maps
		streams += s.streams
		offMap += s.offMap
	}
	return read, streams, offMap
}

// beyond is never negative: a stream reopened after a dropped connection reads nothing.
// A caller that asked for tiles off the map gets no credit at all, however many streams
// it opened — which is what a script buys by opening one before each read. Past
// CertainOffMap such reads the count no longer decides anything: see Watch.
func beyond(maps float64, streams, offMap int) float64 {
	if offMap > 0 {
		return maps
	}
	return math.Max(0, maps-float64(streams))
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
	cutoff := w.clock.Now().Add(-w.config.TrackWindow)

	var unexplained []float64

	w.mu.Lock()

	for scope, c := range w.callers {
		c.prune(cutoff)

		if len(c.slices) == 0 {
			delete(w.callers, scope)
			continue
		}

		if !c.clicked.Before(cutoff) {
			unexplained = append(unexplained, beyond(c.count(cutoff)))
		}
	}

	w.mu.Unlock()

	// Outside the lock: every reader queues on it.
	for _, maps := range unexplained {
		w.onMaps(maps)
	}
}
