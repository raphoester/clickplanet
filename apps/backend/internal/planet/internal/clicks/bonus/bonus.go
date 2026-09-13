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

const (
	KindTripleClicks Kind = "triple_clicks"

	// KindSpreadClicks makes every click take the tiles touching it as well.
	KindSpreadClicks Kind = "spread_clicks"

	// KindBomb grants one bomb, to be dropped within the duration.
	KindBomb Kind = "bomb"
	// KindEncloseClicks makes a click that closes a shape of the caller's own
	// tiles take the tiles inside it as well.
	KindEncloseClicks Kind = "enclose_clicks"
)

// Kinds is every kind this server knows how to grant.
var Kinds = []Kind{KindTripleClicks, KindSpreadClicks, KindBomb, KindEncloseClicks}

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

// Enclosed is a shape an enclose bonus closed, and the tiles it took.
type Enclosed struct {
	CountryID string

	// The click that closed it, which is also one of the wall tiles.
	ClosingTile uint32

	// The caller's tiles that touch the inside.
	Wall []uint32

	// The tiles taken, nearest the closing tile first.
	Filled []uint32

	// Set only on the copy sent to the caller who closed it.
	Yours bool
	Left  int
}

// Event carries exactly one: an Offer reaches its caller, a Taken and an
// Enclosed everyone.
type Event struct {
	Offer    *Offer
	Taken    *Taken
	Enclosed *Enclosed
}

type Reward struct {
	Kind     Kind
	Duration time.Duration

	// For an enclose bonus only: how many shapes, and how big each may be.
	Enclosures        int
	EnclosureMaxTiles int
}

// caller is one scope: its open streams, and the schedule that outlives them.
type caller struct {
	streams map[uint64]chan Event

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	outstanding string
	misses      int

	// Each bonus granted, for MaxBoostPerHour.
	grants []grant
}

type grant struct {
	at       time.Time
	duration time.Duration
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
		entry.grants = append(entry.grants, grant{at: now, duration: offer.duration})

		// A window after the bonus ends, so a second can never land on a running one.
		entry.nextOfferAt = now.Add(offer.duration).Add(r.window())
	}

	reward := Reward{Kind: offer.kind, Duration: offer.duration}
	if offer.kind == KindEncloseClicks {
		reward.Enclosures = r.config.EncloseShapes
		reward.EnclosureMaxTiles = r.config.EncloseMaxTiles
	}

	return reward, true
}

// Dropped brings the next box to a window from now, rather than from when the bomb would have lapsed.
func (r *Registry) Dropped(scope string) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.callers[scope]
	if !ok || entry.outstanding != "" {
		return
	}

	entry.nextOfferAt = minTime(entry.nextOfferAt, now.Add(r.window()))
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}

	return a
}

func (r *Registry) Publish(taken Taken) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.callers {
		entry.send(Event{Taken: &taken})
	}
}

// PublishEnclosed sends a closed shape to everyone. The caller who closed it gets
// a copy of their own, which says so and says how many shapes they have left.
func (r *Registry) PublishEnclosed(scope string, enclosed Enclosed) {
	r.mu.Lock()
	defer r.mu.Unlock()

	theirs := enclosed
	theirs.Yours, theirs.Left = false, 0

	for other, entry := range r.callers {
		if other == scope {
			yours := enclosed
			yours.Yours = true
			entry.send(Event{Enclosed: &yours})

			continue
		}

		entry.send(Event{Enclosed: &theirs})
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
	total := time.Duration(0)
	for _, g := range entry.grants {
		if g.at.After(since) {
			kept = append(kept, g)
			total += g.duration
		}
	}
	entry.grants = kept

	return total >= r.config.MaxBoostPerHour
}

func (r *Registry) offer(scope string, entry *caller, now time.Time) {
	token, err := newToken()
	if err != nil {
		return
	}

	kind := r.drawKind()
	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		Kind:      kind,
		Duration:  r.config.durationOf(kind),
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

// drawKind picks a kind with a chance of its weight over the sum of the weights.
// It walks Kinds rather than the map, so the same draw always lands on the same
// kind.
func (r *Registry) drawKind() Kind {
	total := 0.0
	for _, kind := range Kinds {
		total += r.config.Kinds[kind]
	}

	const resolution = 1 << 53
	n, err := rand.Int(rand.Reader, big.NewInt(resolution))
	if err != nil {
		return KindTripleClicks
	}

	left := float64(n.Int64()) / resolution * total
	last := KindTripleClicks
	for _, kind := range Kinds {
		weight := r.config.Kinds[kind]
		if weight <= 0 {
			continue
		}
		if left < weight {
			return kind
		}
		left -= weight
		last = kind
	}

	// Only float rounding reaches here; it belongs to the last kind with a weight.
	return last
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
