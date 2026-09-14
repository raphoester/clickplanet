// Package ledger remembers, per tile, the last caller who took it and what the tile held before.
//
// It is what the operator tools read to say who is painting what, and to undo one caller's paint.
// It lives in memory only: a restart forgets it, the way it forgets the throttle.
package ledger

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	// How long a take is remembered. Past it, the tile can no longer be traced or reverted.
	Retention     time.Duration
	SweepInterval time.Duration
}

const (
	defaultRetention     = 24 * time.Hour
	defaultSweepInterval = 5 * time.Minute
)

func (c Config) withDefaults() Config {
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = defaultSweepInterval
	}
	return c
}

// Taking is one tile, as its last taker left it.
type Taking struct {
	Tile  uint32
	Scope string
	// Country is what the scope painted; Previous is what the tile held before the scope first took it.
	Country  string
	Previous string
	At       time.Time
}

func New(config Config, clock cptime.Clock) *Ledger {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Ledger{
		config: config.withDefaults(),
		clock:  clock,
		tiles:  make(map[uint32]Taking),
	}
}

type Ledger struct {
	config Config
	clock  cptime.Clock

	mu    sync.Mutex
	tiles map[uint32]Taking
}

// Record notes that scope changed tile from previous to country. A scope that retakes its own tile
// keeps the owner from before its first take, so a revert goes back past all of them.
func (l *Ledger) Record(tile uint32, scope, previous, country string) {
	if scope == "" {
		return
	}

	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if last, ok := l.tiles[tile]; ok && last.Scope == scope {
		previous = last.Previous
	}

	l.tiles[tile] = Taking{Tile: tile, Scope: scope, Country: country, Previous: previous, At: now}
}

// Painted is every remembered take that painted country, whether or not the tile still holds it.
func (l *Ledger) Painted(country string) []Taking {
	return l.collect(func(taking Taking) bool { return taking.Country == country })
}

// TakenBy is every tile scope was the last to take, whether or not the tile still holds it.
func (l *Ledger) TakenBy(scope string) []Taking {
	return l.collect(func(taking Taking) bool { return taking.Scope == scope })
}

func (l *Ledger) collect(keep func(Taking) bool) []Taking {
	l.mu.Lock()
	defer l.mu.Unlock()

	var takings []Taking
	for _, taking := range l.tiles {
		if keep(taking) {
			takings = append(takings, taking)
		}
	}

	return takings
}

// Forget drops these takes, unless the tile was taken again since.
func (l *Ledger) Forget(takings []Taking) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, taking := range takings {
		if l.tiles[taking.Tile] == taking {
			delete(l.tiles, taking.Tile)
		}
	}
}

func (l *Ledger) Run(ctx context.Context) {
	ticker := time.NewTicker(l.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (l *Ledger) sweep() {
	cutoff := l.clock.Now().Add(-l.config.Retention)

	l.mu.Lock()
	defer l.mu.Unlock()

	for tile, taking := range l.tiles {
		if taking.At.Before(cutoff) {
			delete(l.tiles, tile)
		}
	}
}
