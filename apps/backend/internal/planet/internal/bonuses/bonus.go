package bonuses

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kind string

const (
	KindRefill Kind = "refill"

	KindSpreadClicks Kind = "spread_clicks"

	KindBomb          Kind = "bomb"
	KindEncloseClicks Kind = "enclose_clicks"
)

var Kinds = []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks}

type Offer struct {
	Token string

	Seed uint32

	Kind      Kind
	ExpiresAt time.Time
}

type Taken struct {
	CountryID string
	Kind      Kind

	QuizSubject string
}

type Enclosed struct {
	CountryID string

	ClosingTile uint32

	Wall []uint32

	Filled []uint32

	Yours bool
}

type Spread struct {
	CountryID string
	Tile      uint32

	Neighbours []uint32
}

type Event struct {
	Offer    *Offer
	Quiz     *QuizOffer
	Taken    *Taken
	Enclosed *Enclosed
	Spread   *Spread
}

type Reward struct {
	Kind Kind

	Amount int
}

type caller struct {
	streams map[uint64]chan Event

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	players map[Holder]time.Time

	outstanding string
	misses      int

	grants []time.Time

	nextQuizAt      time.Time
	outstandingQuiz string
	quizGrants      []time.Time
}

func (c *caller) watching() bool {
	return len(c.streams) > 0
}

func (c *caller) send(event Event) {
	for _, events := range c.streams {
		select {
		case events <- event:
		default:
		}
	}
}

// Every hook runs with the registry locked, so none may call back into it.
type Report struct {
	Offered func(scope string)

	Lapsed func(scope string)

	Caught func(scope string, after time.Duration)

	Foreign func(scope string)

	QuizOffered func()

	QuizLapsed func(scope string)

	QuizAnswered func(scope string, correct bool, after time.Duration)
}

type Registry struct {
	config   Config
	clock    cptime.Clock
	report   Report
	holdings Holdings
	charges  ChargesConfig

	quizConfig quizzes.Config
	bank       Questions

	mu      sync.Mutex
	callers map[string]*caller
	offers  map[string]*pending

	spent map[string]spentOffer

	quizOffers map[string]*pendingQuiz

	nextID uint64
}

type pending struct {
	scope     string
	kind      Kind
	offeredAt time.Time
	expiresAt time.Time
}

type spentOffer struct {
	scope string
	until time.Time
}

const rememberSpent = 10 * time.Minute

const eventBuffer = 32

type Holdings interface {
	Held(holder Holder) Held
}

func New(config Config, clock cptime.Clock, holdings Holdings) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	config = config.withDefaults()

	return &Registry{
		config:     config,
		charges:    config.ChargesConfig(),
		clock:      clock,
		holdings:   holdings,
		callers:    make(map[string]*caller),
		offers:     make(map[string]*pending),
		spent:      make(map[string]spentOffer),
		quizOffers: make(map[string]*pendingQuiz),
	}
}

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

func (r *Registry) caller(scope string, now time.Time) *caller {
	entry, ok := r.callers[scope]
	if ok {
		return entry
	}

	entry = &caller{
		streams:     make(map[uint64]chan Event),
		players:     make(map[Holder]time.Time),
		nextOfferAt: now.Add(r.window()),
		nextQuizAt:  now.Add(r.quizWindow()),
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

func (r *Registry) Claim(token string, scope string) (Reward, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	offer, ok := r.offers[token]
	if !ok || offer.scope != scope || !now.Before(offer.expiresAt) {
		r.refuse(token, scope)
		return Reward{}, false
	}

	delete(r.offers, token)
	r.spent[token] = spentOffer{scope: scope, until: now.Add(rememberSpent)}

	if r.report.Caught != nil {
		r.report.Caught(scope, now.Sub(offer.offeredAt))
	}

	if entry, known := r.callers[scope]; known {
		entry.outstanding = ""
		entry.misses = 0
		entry.grants = append(entry.grants, now)

		entry.nextOfferAt = now.Add(r.window())
	}

	return Reward{Kind: offer.kind, Amount: r.amountOf(offer.kind)}, true
}

func (r *Registry) refuse(token string, scope string) {
	owner := ""
	if offer, ok := r.offers[token]; ok {
		owner = offer.scope
	} else if spent, ok := r.spent[token]; ok {
		owner = spent.scope
	}

	if owner != scope && r.report.Foreign != nil {
		r.report.Foreign(scope)
	}
}

func (r *Registry) Publish(taken Taken) {
	r.broadcast(Event{Taken: &taken})
}

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
	r.forgetSpent(now)
	r.sweepQuizzes(now)
	r.forgetStale(now)

	for scope, entry := range r.callers {
		if !r.due(entry, now) {
			continue
		}

		kinds := r.offerable(entry, now)
		if kinds.Empty() {
			entry.nextOfferAt = now.Add(r.window())
			continue
		}

		r.offer(scope, entry, now, kinds)
	}
}

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
	if r.report.Offered != nil {
		r.report.Offered(scope)
	}
}

func (r *Registry) collectMisses(now time.Time) {
	for token, offer := range r.offers {
		if now.Before(offer.expiresAt) {
			continue
		}

		delete(r.offers, token)
		r.spent[token] = spentOffer{scope: offer.scope, until: now.Add(rememberSpent)}

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

func (r *Registry) forgetSpent(now time.Time) {
	for token, spent := range r.spent {
		if now.After(spent.until) {
			delete(r.spent, token)
		}
	}
}

func (r *Registry) forgetStale(now time.Time) {
	for scope, entry := range r.callers {
		if entry.watching() || now.Sub(entry.lastSeen) <= r.config.ForgetAfter {
			continue
		}

		if entry.outstandingQuiz != "" {
			delete(r.quizOffers, entry.outstandingQuiz)
		}

		delete(r.callers, scope)
	}
}

func (r *Registry) window() time.Duration {
	return drawWindow(r.config.MinInterval, r.config.MaxInterval)
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

	return last
}

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
