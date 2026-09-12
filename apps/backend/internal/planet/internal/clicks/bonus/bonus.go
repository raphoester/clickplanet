// Package bonus hands out the question-mark boxes that fly past the planet:
// addressed to one caller, on a schedule of that caller's own.
package bonus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kind string

const KindTripleClicks Kind = "triple_clicks"

type Offer struct {
	Token string

	// Names the flight path; every client draws the same orbit from it.
	Seed uint32

	Kind      Kind
	Duration  time.Duration
	ExpiresAt time.Time
}

type Taken struct {
	CountryID string
	Kind      Kind
}

// Event carries exactly one: an Offer reaches its caller, a Taken everyone.
type Event struct {
	Offer *Offer
	Taken *Taken
}

type Reward struct {
	Kind     Kind
	Duration time.Duration
}

// caller is one scope: its open streams, and the schedule that outlives them.
type caller struct {
	streams map[uint64]chan Event

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	outstanding string
	misses      int

	// When each bonus was granted, for MaxBoostPerHour.
	grants []time.Time
}

func (c *caller) watching() bool {
	return len(c.streams) > 0
}

// send drops rather than blocks, as the tile fanout does for a slow subscriber.
func (c *caller) send(event Event) {
	for _, events := range c.streams {
		select {
		case events <- event:
		default:
		}
	}
}

// Report is told what the sweep did, so the counters live at the edge and this
// package keeps knowing nothing about Prometheus.
type Report struct {
	Offered func()
	Lapsed  func()
}

type Registry struct {
	config Config
	clock  cptime.Clock
	report Report

	mu      sync.Mutex
	callers map[string]*caller
	offers  map[string]*pending
	nextID  uint64
}

type pending struct {
	scope     string
	kind      Kind
	duration  time.Duration
	expiresAt time.Time
}

const eventBuffer = 8

func New(config Config, clock cptime.Clock) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Registry{
		config:  config.withDefaults(),
		clock:   clock,
		callers: make(map[string]*caller),
		offers:  make(map[string]*pending),
	}
}

// Observe attaches the counters. Optional, so a test builds a registry without
// touching a metrics registry.
func (r *Registry) Observe(report Report) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report = report
}

func (r *Registry) counted(hook func()) {
	if hook != nil {
		hook()
	}
}

// Attend adds a stream to the feed; the returned func must be called when it ends.
func (r *Registry) Attend(scope string) (<-chan Event, func()) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(scope, now)

	r.nextID++
	id := r.nextID

	events := make(chan Event, eventBuffer)
	entry.streams[id] = events
	entry.lastSeen = now

	return events, func() { r.leave(scope, id) }
}

// caller keeps a schedule across a disconnect, so reloading cannot reroll it.
func (r *Registry) caller(scope string, now time.Time) *caller {
	entry, ok := r.callers[scope]
	if ok {
		return entry
	}

	entry = &caller{
		streams:     make(map[uint64]chan Event),
		nextOfferAt: now.Add(r.window()),
		lastSeen:    now,
	}
	r.callers[scope] = entry

	return entry
}

func (r *Registry) leave(scope string, id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.callers[scope]
	if !ok {
		return
	}

	delete(entry.streams, id)
	entry.lastSeen = r.clock.Now()
}

// Clicked marks a caller as playing. Boxes only go to callers who are.
func (r *Registry) Clicked(scope string) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.caller(scope, now).lastClickAt = now
}

// Claim fails for a token unknown, spent, lapsed, or offered to somebody else.
func (r *Registry) Claim(token string, scope string) (Reward, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	offer, ok := r.offers[token]
	if !ok || offer.scope != scope || !now.Before(offer.expiresAt) {
		return Reward{}, false
	}

	delete(r.offers, token)

	if entry, known := r.callers[scope]; known {
		entry.outstanding = ""
		entry.misses = 0
		entry.grants = append(entry.grants, now)

		// A window after the bonus ends, so a second can never land on a running one.
		entry.nextOfferAt = now.Add(offer.duration).Add(r.window())
	}

	return Reward{Kind: offer.kind, Duration: offer.duration}, true
}

func (r *Registry) Publish(taken Taken) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.callers {
		entry.send(Event{Taken: &taken})
	}
}

func (r *Registry) Multiplier() float64 {
	return r.config.Multiplier
}

func (r *Registry) Run(ctx context.Context) {
	ticker := time.NewTicker(r.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Registry) sweep() {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.collectMisses(now)
	r.forgetStale(now)

	for scope, entry := range r.callers {
		if r.due(entry, now) {
			r.offer(scope, entry, now)
		}
	}
}

// due loses the slot for a caller who was away, rather than banking it.
func (r *Registry) due(entry *caller, now time.Time) bool {
	if !entry.watching() || entry.outstanding != "" || now.Before(entry.nextOfferAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > r.config.ActiveWithin || r.capped(entry, now) {
		entry.nextOfferAt = now.Add(r.window())
		return false
	}

	return true
}

func (r *Registry) capped(entry *caller, now time.Time) bool {
	since := now.Add(-time.Hour)

	kept := entry.grants[:0]
	for _, at := range entry.grants {
		if at.After(since) {
			kept = append(kept, at)
		}
	}
	entry.grants = kept

	return time.Duration(len(kept))*r.config.Duration >= r.config.MaxBoostPerHour
}

func (r *Registry) offer(scope string, entry *caller, now time.Time) {
	token, err := newToken()
	if err != nil {
		return
	}

	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		Kind:      KindTripleClicks,
		Duration:  r.config.Duration,
		ExpiresAt: now.Add(r.config.OfferTTL),
	}

	r.offers[token] = &pending{
		scope:     scope,
		kind:      offer.Kind,
		duration:  offer.Duration,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstanding = token
	entry.nextOfferAt = offer.ExpiresAt.Add(r.window())

	entry.send(Event{Offer: &offer})
	r.counted(r.report.Offered)
}

// collectMisses retires unspent tokens, bringing the next box forward once.
func (r *Registry) collectMisses(now time.Time) {
	for token, offer := range r.offers {
		if now.Before(offer.expiresAt) {
			continue
		}

		delete(r.offers, token)

		entry, ok := r.callers[offer.scope]
		if !ok || entry.outstanding != token {
			continue
		}

		entry.outstanding = ""
		if entry.misses == 0 {
			entry.nextOfferAt = now.Add(r.config.MissRetry)
		}
		entry.misses++
		r.counted(r.report.Lapsed)
	}
}

func (r *Registry) forgetStale(now time.Time) {
	for scope, entry := range r.callers {
		if entry.watching() || now.Sub(entry.lastSeen) <= r.config.ForgetAfter {
			continue
		}

		delete(r.callers, scope)
	}
}

func (r *Registry) window() time.Duration {
	spread := r.config.MaxInterval - r.config.MinInterval
	if spread <= 0 {
		return r.config.MinInterval
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(spread)))
	if err != nil {
		return r.config.MinInterval
	}

	return r.config.MinInterval + time.Duration(n.Int64())
}

func newToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes for a bonus token: %w", err)
	}

	return hex.EncodeToString(raw), nil
}

func randomSeed() uint32 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<32))
	if err != nil {
		return 0
	}

	return uint32(n.Int64())
}
