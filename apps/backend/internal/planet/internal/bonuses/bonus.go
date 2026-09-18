// Package bonus hands out the question-mark boxes that fly past the planet:
// addressed to one caller, on a schedule of that caller's own.
package bonuses

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kind string

// Every kind is a charge: held until it is spent (see Hand), one of each at most.
const (
	// KindRefill fills the caller's click bank, when the caller chooses.
	KindRefill Kind = "refill"

	// KindSpreadClicks makes every click take the tiles touching it as well.
	KindSpreadClicks Kind = "spread_clicks"

	// KindBomb grants one bomb, kept until it is dropped.
	KindBomb Kind = "bomb"
	// KindEncloseClicks makes a click that closes a shape of the caller's own
	// tiles take the tiles inside it as well, once.
	KindEncloseClicks Kind = "enclose_clicks"
)

// Kinds is every kind this server knows how to grant.
var Kinds = []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks}

type Offer struct {
	Token string

	// Names the flight path; every client draws the same orbit from it.
	Seed uint32

	Kind      Kind
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
}

// Spread is a click a spread bonus carried onto the tiles touching it.
type Spread struct {
	CountryID string
	Tile      uint32

	// The neighbours the click also took. Empty for a lone island.
	Neighbours []uint32
}

// Event carries exactly one: an Offer reaches its caller, the rest everyone.
type Event struct {
	Offer    *Offer
	Taken    *Taken
	Enclosed *Enclosed
	Spread   *Spread
}

type Reward struct {
	Kind Kind

	// How much the box gives: enclosures or spread clicks, drawn at the claim. One for a refill or a bomb.
	Amount int
}

// caller is one scope: its open streams, and the schedule that outlives them.
type caller struct {
	streams map[uint64]chan Event

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	// Which accounts clicked from this scope, and when last: whose charges keep a kind from being offered.
	players map[Holder]time.Time

	outstanding string
	misses      int

	// When each charge was granted, for MaxChargesPerHour.
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

// Report is told what happened to each box, so the counters and the antibot
// live at the edge and this package keeps knowing nothing about either. Every
// hook is optional, and each is called with the registry locked, so none may
// call back into it.
type Report struct {
	Offered func()

	// Lapsed is a box its caller never claimed.
	Lapsed func(scope string)

	// Caught is a box claimed, and how long after it was offered.
	Caught func(scope string, after time.Duration)
}

type Registry struct {
	config   Config
	clock    cptime.Clock
	report   Report
	holdings Holdings
	// What a spread pool holds when full, so a full pool is not offered another box.
	charges ChargesConfig

	mu      sync.Mutex
	callers map[string]*caller
	offers  map[string]*pending
	nextID  uint64
}

type pending struct {
	scope     string
	kind      Kind
	offeredAt time.Time
	expiresAt time.Time
}

// A stream gets an event per spread click anyone makes, a few a second each, so
// this is sized for a burst of those rather than for the rare offer.
const eventBuffer = 32

// Holdings is what a holder has in hand, which the schedule reads so nobody is offered a second of a kind.
type Holdings interface {
	Held(holder Holder) Held
}

func New(config Config, clock cptime.Clock, holdings Holdings) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	config = config.withDefaults()

	return &Registry{
		config:   config,
		charges:  config.ChargesConfig(),
		clock:    clock,
		holdings: holdings,
		callers:  make(map[string]*caller),
		offers:   make(map[string]*pending),
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
		players:     make(map[Holder]time.Time),
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

// Clicked marks a caller as playing, and holder as one of the players behind it. Boxes only go to callers
// who are, and not in a kind any of its players holds.
func (r *Registry) Clicked(scope string, holder Holder) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(scope, now)
	entry.lastClickAt = now
	if holder != NoHolder {
		entry.players[holder] = now
	}
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

	if r.report.Caught != nil {
		r.report.Caught(scope, now.Sub(offer.offeredAt))
	}

	if entry, known := r.callers[scope]; known {
		entry.outstanding = ""
		entry.misses = 0
		entry.grants = append(entry.grants, now)

		// A charge has no end, so the next is due a window after the claim: a held bomb does not hold
		// back every other box.
		entry.nextOfferAt = now.Add(r.window())
	}

	return Reward{Kind: offer.kind, Amount: r.amountOf(offer.kind)}, true
}

func (r *Registry) Publish(taken Taken) {
	r.broadcast(Event{Taken: &taken})
}

// PublishEnclosed sends a closed shape to everyone. The caller who closed it gets
// a copy of their own, which says so.
func (r *Registry) PublishEnclosed(scope string, enclosed Enclosed) {
	r.mu.Lock()
	defer r.mu.Unlock()

	theirs := enclosed
	theirs.Yours = false

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

// PublishSpread sends a spread click to everyone, so every client can show it.
func (r *Registry) PublishSpread(spread Spread) {
	r.broadcast(Event{Spread: &spread})
}

func (r *Registry) broadcast(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.callers {
		entry.send(event)
	}
}

func (r *Registry) Name() string { return "bonus-boxes" }

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
		if !r.due(entry, now) {
			continue
		}

		kinds := r.offerable(entry, now)
		if kinds.Empty() {
			// Nothing it may be given: the slot is lost, as it is for a caller who was away.
			entry.nextOfferAt = now.Add(r.window())
			continue
		}

		r.offer(scope, entry, now, kinds)
	}
}

// due loses the slot for a caller who was away, rather than banking it.
func (r *Registry) due(entry *caller, now time.Time) bool {
	if !entry.watching() || entry.outstanding != "" || now.Before(entry.nextOfferAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > r.config.ActiveWithin {
		entry.nextOfferAt = now.Add(r.window())
		return false
	}

	return true
}

// offerable is every kind with a weight that the caller may be given now. A kind any player who clicked
// from this scope within ActiveWithin holds is left out, so nobody holds two of one kind, and so is spread
// when a player's pool is already full. Past
// MaxChargesPerHour charges in the hour nothing is, and a scope where no account is playing is offered
// nothing: only an account can hold a charge. It forgets the players who stopped clicking on the way.
func (r *Registry) offerable(entry *caller, now time.Time) *cpcolls.Set[Kind] {
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))

	held := cpcolls.NewSet[Kind]()
	for holder, clicked := range entry.players {
		if now.Sub(clicked) > r.config.ActiveWithin {
			delete(entry.players, holder)
			continue
		}
		held.Add(r.holdings.Held(holder).Full(r.charges)...)
	}

	if len(entry.players) == 0 || r.grantedWithinTheHour(entry, now) >= r.config.MaxChargesPerHour {
		return kinds
	}

	for _, kind := range Kinds {
		if r.config.Kinds[kind] > 0 && !held.Contains(kind) {
			kinds.Add(kind)
		}
	}

	return kinds
}

// grantedWithinTheHour is how many charges were granted in the last hour. It forgets the older grants on
// the way.
func (r *Registry) grantedWithinTheHour(entry *caller, now time.Time) int {
	since := now.Add(-time.Hour)

	kept := entry.grants[:0]
	for _, at := range entry.grants {
		if at.After(since) {
			kept = append(kept, at)
		}
	}
	entry.grants = kept

	return len(kept)
}

func (r *Registry) offer(scope string, entry *caller, now time.Time, kinds *cpcolls.Set[Kind]) {
	token, err := newToken()
	if err != nil {
		return
	}

	kind := r.drawKind(kinds)
	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		Kind:      kind,
		ExpiresAt: now.Add(r.config.OfferTTL),
	}

	r.offers[token] = &pending{
		scope:     scope,
		kind:      offer.Kind,
		offeredAt: now,
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
		if r.report.Lapsed != nil {
			r.report.Lapsed(offer.scope)
		}
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

// drawKind picks one of kinds with a chance of its weight over the sum of their weights.
// It walks Kinds rather than the map, so the same draw always lands on the same
// kind. kinds is never empty, and holds only kinds with a weight.
func (r *Registry) drawKind(kinds *cpcolls.Set[Kind]) Kind {
	total := 0.0
	var last Kind
	for _, kind := range Kinds {
		if kinds.Contains(kind) {
			total += r.config.Kinds[kind]
			last = kind
		}
	}

	const resolution = 1 << 53
	n, err := rand.Int(rand.Reader, big.NewInt(resolution))
	if err != nil {
		return last
	}

	left := float64(n.Int64()) / resolution * total
	for _, kind := range Kinds {
		weight := r.config.Kinds[kind]
		if !kinds.Contains(kind) {
			continue
		}
		if left < weight {
			return kind
		}
		left -= weight
	}

	// Only float rounding reaches here; it belongs to the last kind with a weight.
	return last
}

// amountOf draws how much a box of kind gives: 1 to MaxPerBox enclosures or spread clicks, uniformly.
func (r *Registry) amountOf(kind Kind) int {
	most := 1
	switch kind {
	case KindSpreadClicks:
		most = r.config.Spread.MaxPerBox
	case KindEncloseClicks:
		most = r.config.Enclose.MaxPerBox
	case KindRefill, KindBomb:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(most)))
	if err != nil {
		return 1
	}

	return 1 + int(n.Int64())
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
