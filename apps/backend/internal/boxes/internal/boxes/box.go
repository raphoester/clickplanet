package boxes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Offer is a box sent down one caller's streams. It says nothing of what it holds: the bonuses module draws that when
// it is caught.
type Offer struct {
	Token string

	Seed uint32

	ExpiresAt time.Time
}

// Caught is a box claimed by the caller it was offered to.
type Caught struct {
	// Welcome is a newcomer's early box.
	Welcome bool
}

// Click is a click planet took, struck a shield with or found its tile already wearing its flag, as planet's ClickTried
// tells it: what it says of its caller, and when.
type Click struct {
	Caller Caller

	// At is when it was tried. Zero is when it is heard.
	At time.Time

	// Born is when the account was made, read from its id.
	Born time.Time
}

type caller struct {
	streams map[uint64]chan Offer

	scope string

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	players map[Account]time.Time

	outstanding string
	misses      int

	caught []time.Time

	owedWelcome bool
}

func (c *caller) watching() bool {
	return len(c.streams) > 0
}

func (c *caller) send(offer Offer) {
	for _, offers := range c.streams {
		select {
		case offers <- offer:
		default:
		}
	}
}

// Every hook runs with the registry locked, so none may call back into it.
type Report struct {
	Offered func()

	Lapsed func(scope string)

	Caught func(scope string, after time.Duration)

	Foreign func(scope string)
}

type Pace struct {
	MinInterval time.Duration
	MaxInterval time.Duration

	MaxBoxesPerHour int
}

type Registry struct {
	config Config
	rules  func() Rules
	clock  cptime.Clock
	report Report
	room   Room

	mu           sync.Mutex
	followedPace Pace
	callers      map[Entrant]*caller
	offers       map[string]*pending

	spent map[string]spentOffer

	newcomers map[Account]time.Time

	nextID uint64
}

type pending struct {
	entrant   Entrant
	scope     string
	welcome   bool
	offeredAt time.Time
	expiresAt time.Time
}

type spentOffer struct {
	entrant Entrant
	until   time.Time
}

const rememberSpent = 10 * time.Minute

const offerBuffer = 32

func New(config Config, clock cptime.Clock, room Room, rules func() Rules) *Registry {
	return &Registry{
		config:       config.withDefaults(),
		rules:        rules,
		clock:        clock,
		room:         room,
		followedPace: rules().Pace,
		callers:      make(map[Entrant]*caller),
		offers:       make(map[string]*pending),
		spent:        make(map[string]spentOffer),
		newcomers:    make(map[Account]time.Time),
	}
}

func (r *Registry) Observe(report Report) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report = report
}

// follow brings in a box due later than the longest wait of a new pace.
func (r *Registry) follow(rules Rules, now time.Time) {
	if rules.Pace == r.followedPace {
		return
	}

	for _, entry := range r.callers {
		if entry.nextOfferAt.After(now.Add(rules.Pace.MaxInterval)) {
			entry.nextOfferAt = now.Add(boxWait(rules))
		}
	}
	r.followedPace = rules.Pace
}

func (r *Registry) counted(hook func()) {
	if hook != nil {
		hook()
	}
}

func (r *Registry) Attend(entrant Entrant) (<-chan Offer, func()) {
	now := r.clock.Now()
	rules := r.rules()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(entrant, rules, now)

	r.nextID++
	id := r.nextID

	offers := make(chan Offer, offerBuffer)
	entry.streams[id] = offers
	entry.lastSeen = now

	return offers, func() { r.leave(entrant, id) }
}

func (r *Registry) caller(entrant Entrant, rules Rules, now time.Time) *caller {
	entry, ok := r.callers[entrant]
	if ok {
		return entry
	}

	entry = &caller{
		streams:     make(map[uint64]chan Offer),
		players:     make(map[Account]time.Time),
		nextOfferAt: now.Add(boxWait(rules)),
		lastSeen:    now,
	}
	r.callers[entrant] = entry

	return entry
}

func (r *Registry) leave(entrant Entrant, id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.callers[entrant]
	if !ok {
		return
	}

	delete(entry.streams, id)
	entry.lastSeen = r.clock.Now()
}

// Clicked hears a click a moment after it was tried; one tried longer ago than activeWithin, heard late after a boot,
// is no sign of a player.
func (r *Registry) Clicked(click Click) {
	now := r.clock.Now()
	rules := r.rules()

	at := click.At
	if at.IsZero() || at.After(now) {
		at = now
	}
	if now.Sub(at) > rules.ActiveWithin {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(EntrantOf(click.Caller), rules, now)
	entry.scope = click.Caller.Scope
	if at.After(entry.lastClickAt) {
		entry.lastClickAt = at
	}
	if click.Caller.Account == NoAccount {
		return
	}

	if at.After(entry.players[click.Caller.Account]) {
		entry.players[click.Caller.Account] = at
	}
	r.welcome(entry, click, rules, now)
}

func (r *Registry) Claim(token string, entrant Entrant, scope string) (Caught, bool) {
	now := r.clock.Now()
	rules := r.rules()

	r.mu.Lock()
	defer r.mu.Unlock()

	offer, ok := r.offers[token]
	if !ok || offer.entrant != entrant || !now.Before(offer.expiresAt) {
		r.refuse(token, entrant, scope)
		return Caught{}, false
	}

	delete(r.offers, token)
	r.spent[token] = spentOffer{entrant: entrant, until: now.Add(rememberSpent)}

	if r.report.Caught != nil {
		r.report.Caught(scope, now.Sub(offer.offeredAt))
	}

	if entry, known := r.callers[entrant]; known {
		entry.outstanding = ""
		entry.misses = 0
		entry.caught = append(entry.caught, now)

		entry.nextOfferAt = now.Add(boxWait(rules))
	}

	return Caught{Welcome: offer.welcome}, true
}

func (r *Registry) refuse(token string, entrant Entrant, scope string) {
	owner := Entrant("")
	if offer, ok := r.offers[token]; ok {
		owner = offer.entrant
	} else if spent, ok := r.spent[token]; ok {
		owner = spent.entrant
	}

	if owner != entrant && r.report.Foreign != nil {
		r.report.Foreign(scope)
	}
}

func (r *Registry) Name() string { return "bonus-boxes" }

func (r *Registry) Run(ctx context.Context) {
	ticker := time.NewTicker(r.config.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.sweep(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// sweep decides who is due under the lock, asks the wallet for their room outside it, then offers under it again to
// whoever is due still: the room is a call out of the boxes, and nothing else waits on the registry while it runs.
func (r *Registry) sweep(ctx context.Context) {
	now := r.clock.Now()
	rules := r.rules()

	due := r.dueCallers(rules, now)
	r.ask(ctx, due, rules)
	r.offerDue(due, rules, now)
}

// dueOne is a caller found due, the accounts that played behind it then, and what the wallet said of their room.
type dueOne struct {
	entrant  Entrant
	accounts []Account

	room bool
	err  error
}

// dueCallers are the callers due a box, once the registry is tidied under the same lock.
func (r *Registry) dueCallers(rules Rules, now time.Time) []*dueOne {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tidy(rules, now)

	var due []*dueOne
	for entrant, entry := range r.callers {
		if r.due(entry, rules, now) {
			due = append(due, &dueOne{entrant: entrant, accounts: entry.playing(rules, now)})
		}
	}

	return due
}

func (r *Registry) tidy(rules Rules, now time.Time) {
	r.follow(rules, now)
	r.collectMisses(rules, now)
	r.forgetSpent(now)
	r.forgetStale(now)
	r.forgetNewcomers(now)
}

// ask runs with the registry unlocked. A frozen set offers nothing, so it asks nothing.
func (r *Registry) ask(ctx context.Context, due []*dueOne, rules Rules) {
	if rules.Frozen {
		return
	}
	for _, one := range due {
		if len(one.accounts) > 0 {
			one.room, one.err = r.room.Any(ctx, one.accounts)
		}
	}
}

func (r *Registry) offerDue(due []*dueOne, rules Rules, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, one := range due {
		entry, ok := r.callers[one.entrant]
		if !ok || !entry.playedBy(one.accounts, rules, now) || !r.due(entry, rules, now) {
			continue
		}

		welcome := entry.owedWelcome
		entry.owedWelcome = false
		if !r.offerable(entry, one, rules, now) {
			entry.nextOfferAt = now.Add(boxWait(rules))
			continue
		}

		r.offer(one.entrant, entry, rules, now, welcome)
	}
}

// playing forgets the accounts that last clicked longer ago than activeWithin, and is the others.
func (c *caller) playing(rules Rules, now time.Time) []Account {
	accounts := make([]Account, 0, len(c.players))
	for account, clicked := range c.players {
		if now.Sub(clicked) > rules.ActiveWithin {
			delete(c.players, account)
			continue
		}
		accounts = append(accounts, account)
	}

	return accounts
}

// playedBy is true while every account playing behind the caller is one the wallet was asked of: one that clicked
// first while the room was asked waits for the next sweep, rather than be offered on a room it was not counted in.
func (c *caller) playedBy(asked []Account, rules Rules, now time.Time) bool {
	for account, clicked := range c.players {
		if now.Sub(clicked) <= rules.ActiveWithin && !slices.Contains(asked, account) {
			return false
		}
	}

	return true
}

func (r *Registry) due(entry *caller, rules Rules, now time.Time) bool {
	if !entry.watching() || entry.outstanding != "" || now.Before(entry.nextOfferAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > rules.ActiveWithin {
		entry.nextOfferAt = now.Add(boxWait(rules))
		return false
	}

	return true
}

// offerable is false while the wallet said the hands playing behind the caller have room for no kind at all: a box
// would pay nothing. Which kind it holds is the bonuses module's to draw.
func (r *Registry) offerable(entry *caller, one *dueOne, rules Rules, now time.Time) bool {
	if one.err != nil || rules.Frozen {
		// Offered nothing: the slot is lost, as an away player's.
		return false
	}

	if len(one.accounts) == 0 || r.caughtWithinTheHour(entry, now) >= rules.Pace.MaxBoxesPerHour {
		return false
	}

	return one.room
}

func (r *Registry) caughtWithinTheHour(entry *caller, now time.Time) int {
	since := now.Add(-time.Hour)

	kept := entry.caught[:0]
	for _, at := range entry.caught {
		if at.After(since) {
			kept = append(kept, at)
		}
	}
	entry.caught = kept

	return len(kept)
}

func (r *Registry) offer(entrant Entrant, entry *caller, rules Rules, now time.Time, welcome bool) {
	token, err := newToken()
	if err != nil {
		return
	}

	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		ExpiresAt: now.Add(rules.OfferTTL),
	}

	r.offers[token] = &pending{
		entrant:   entrant,
		scope:     entry.scope,
		welcome:   welcome,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstanding = token
	entry.nextOfferAt = offer.ExpiresAt.Add(boxWait(rules))

	entry.send(offer)
	r.counted(r.report.Offered)
}

func (r *Registry) collectMisses(rules Rules, now time.Time) {
	for token, offer := range r.offers {
		if now.Before(offer.expiresAt) {
			continue
		}

		delete(r.offers, token)
		r.spent[token] = spentOffer{entrant: offer.entrant, until: now.Add(rememberSpent)}

		entry, ok := r.callers[offer.entrant]
		if !ok || entry.outstanding != token {
			continue
		}

		entry.outstanding = ""
		if entry.misses == 0 {
			entry.nextOfferAt = now.Add(rules.MissRetry)
		}
		entry.misses++
		if r.report.Lapsed != nil {
			r.report.Lapsed(offer.scope)
		}
	}
}

func (r *Registry) forgetSpent(now time.Time) {
	for token, spent := range r.spent {
		if now.After(spent.until) {
			delete(r.spent, token)
		}
	}
}

func (r *Registry) forgetStale(now time.Time) {
	for entrant, entry := range r.callers {
		if entry.watching() || now.Sub(entry.lastSeen) <= r.config.ForgetAfter {
			continue
		}

		delete(r.callers, entrant)
	}
}

func boxWait(rules Rules) time.Duration {
	return drawWindow(rules.Pace.MinInterval, rules.Pace.MaxInterval)
}

func drawWindow(shortest, longest time.Duration) time.Duration {
	spread := longest - shortest
	if spread <= 0 {
		return shortest
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(spread)))
	if err != nil {
		return shortest
	}

	return shortest + time.Duration(n.Int64())
}

func newToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes for a box token: %w", err)
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
