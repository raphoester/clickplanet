// Package bonus hands out the question-mark boxes that fly past the planet, and
// decides who gets one.
//
// **The server picks the winner, and it is not a race.** A box broadcast to
// everyone would be caught by whichever client reacts fastest, and that is
// always a script: it reads the event off the stream and answers in twenty
// milliseconds while a person is still moving the mouse. This would then be a
// machine for handing extra clicks to exactly the callers internal/antibot
// exists to stop.
//
// So an offer is **addressed**. One attendee is drawn at random, the box is sent
// down that one stream, and nobody else can see it or claim it. Reflexes buy
// nothing; the only thing left to do is notice the box and click it, which is
// the part that was supposed to be the game.
//
// What everyone *does* see is the catch, once it has happened. A private reward
// with a public outcome keeps the spectacle without the scramble.
package bonus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// Kind is what a box is worth. One today; the type exists because the wire
// already carries an enum and a second kind must not be a second code path.
type Kind string

const KindTripleClicks Kind = "triple_clicks"

// Offer is a box put in front of one attendee.
type Offer struct {
	// Unguessable, single use, and only good for the scope it was sent to.
	Token string

	// Names the flight path. Every client draws the same orbit from the same
	// number, so this travels instead of a trajectory.
	Seed uint32

	Kind      Kind
	Duration  time.Duration
	ExpiresAt time.Time
}

// Taken is the public half: somebody caught one.
type Taken struct {
	CountryID string
	Kind      Kind
}

// Event is what an attendee's stream carries. Exactly one field is set — an
// Offer reaches only the attendee it was drawn for, a Taken reaches everyone.
type Event struct {
	Offer *Offer
	Taken *Taken
}

// Reward is what a redeemed token is worth to the caller that redeemed it.
type Reward struct {
	Kind     Kind
	Duration time.Duration
}

// attendee is one caller that is currently watching the planet.
//
// Keyed on the caller's scope rather than on the connection, so twenty tabs are
// one entry and not twenty tickets in the draw — but each of those tabs holds
// its **own channel**, and a box goes to all of them.
//
// One channel shared between the tabs would be delivered to whichever goroutine
// happened to win the receive, so the box would appear in a tab at random —
// including one the player is not looking at.
type attendee struct {
	streams map[uint64]chan Event
}

func (a *attendee) send(event Event) {
	for _, events := range a.streams {
		select {
		case events <- event:
		default:
			// This tab is not reading. It costs the draw nothing, exactly as a
			// slow tile subscriber drops updates rather than stalling the fanout.
		}
	}
}

type Registry struct {
	config Config
	clock  xtime.Provider

	mu        sync.Mutex
	attendees map[string]*attendee
	offers    map[string]*pending
	nextID    uint64
}

type pending struct {
	scope     string
	kind      Kind
	duration  time.Duration
	expiresAt time.Time
}

// The buffer is per attendee. An offer nobody is reading is an offer that is
// dropped rather than one that blocks the draw, exactly as a slow subscriber
// drops tile updates rather than stalling the fanout.
const eventBuffer = 8

func New(config Config, clock xtime.Provider) *Registry {
	if clock == nil {
		clock = xtime.ActualProvider{}
	}

	return &Registry{
		config:    config.withDefaults(),
		clock:     clock,
		attendees: make(map[string]*attendee),
		offers:    make(map[string]*pending),
	}
}

// Attend puts a caller in the draw for as long as its stream is open, and hands
// back the channel its boxes arrive on.
//
// The returned cancel must be called when the stream ends. There is no context
// goroutine here on purpose: the caller is a streaming handler that already has
// a `defer`, and one goroutine per connected client to watch a context is a cost
// the fanout deliberately does not pay.
func (r *Registry) Attend(scope string) (<-chan Event, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.attendees[scope]
	if !ok {
		entry = &attendee{streams: make(map[uint64]chan Event)}
		r.attendees[scope] = entry
	}

	r.nextID++
	id := r.nextID

	events := make(chan Event, eventBuffer)
	entry.streams[id] = events

	return events, func() { r.leave(scope, id) }
}

func (r *Registry) leave(scope string, id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.attendees[scope]
	if !ok {
		return
	}

	delete(entry.streams, id)
	if len(entry.streams) == 0 {
		delete(r.attendees, scope)
	}
}

// Offer draws one attendee and puts a box in front of them.
//
// It reports whether anything was offered: with nobody watching there is nobody
// to draw, and a box offered to an empty room would only be a token to sweep up
// later.
func (r *Registry) Offer() (Offer, bool) {
	now := r.clock.Now()

	token, err := newToken()
	if err != nil {
		// The only failure is the system's entropy source, and a box is not
		// worth failing anything over: skip this round.
		return Offer{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.forgetLapsed(now)

	scope, entry, ok := r.draw()
	if !ok {
		return Offer{}, false
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

	// Every tab this caller has open, and no other caller's. Nothing is retried
	// into a second attendee if they are not reading: the draw has happened,
	// and the token simply lapses.
	entry.send(Event{Offer: &offer})

	return offer, true
}

// draw picks one attendee uniformly. Uniformly and not by how long they have
// been connected: rewarding a long connection rewards leaving a tab open, which
// is the opposite of the thing this is meant to reward.
func (r *Registry) draw() (string, *attendee, bool) {
	if len(r.attendees) == 0 {
		return "", nil, false
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(r.attendees))))
	if err != nil {
		return "", nil, false
	}

	// Go randomises map iteration order, but not uniformly enough to be the
	// draw itself; the index is what decides, and the walk only reaches it.
	wanted := int(n.Int64())
	for scope, entry := range r.attendees {
		if wanted == 0 {
			return scope, entry, true
		}
		wanted--
	}

	return "", nil, false
}

// Claim redeems a token for the caller holding it.
//
// It fails for a token that was never offered, one already redeemed, one that
// has lapsed, and — the one that matters — one offered to somebody else. A
// token lifted off another caller's stream is worth nothing here.
func (r *Registry) Claim(token string, scope string) (Reward, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.forgetLapsed(now)

	offer, ok := r.offers[token]
	if !ok || offer.scope != scope {
		return Reward{}, false
	}

	// Taken off the moment it is spent, which is what makes it single use.
	delete(r.offers, token)

	return Reward{Kind: offer.kind, Duration: offer.duration}, true
}

// Publish tells every attendee that a box was caught.
func (r *Registry) Publish(taken Taken) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.attendees {
		entry.send(Event{Taken: &taken})
	}
}

// Multiplier is what a reward multiplies the click allowance by.
func (r *Registry) Multiplier() float64 {
	return r.config.Multiplier
}

// Run offers a box every interval for as long as the process lives.
func (r *Registry) Run(ctx context.Context) {
	ticker := time.NewTicker(r.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.Offer()
		case <-ctx.Done():
			return
		}
	}
}

// forgetLapsed drops the tokens that can no longer be spent. Called from the
// two paths that already hold the lock, so an offer nobody claimed costs no
// timer of its own.
func (r *Registry) forgetLapsed(now time.Time) {
	for token, offer := range r.offers {
		if !now.Before(offer.expiresAt) {
			delete(r.offers, token)
		}
	}
}

func newToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
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
