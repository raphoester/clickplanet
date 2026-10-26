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

const (
	KindRefill Kind = "refill"

	KindSpreadClicks Kind = "spread_clicks"

	KindBomb          Kind = "bomb"
	KindEncloseClicks Kind = "enclose_clicks"

	KindShields Kind = "shields"
)

var Kinds = []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields}

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

type Click struct {
	Entrant Entrant
	Scope   string
	Holder  Holder

	Born time.Time

	Dry bool
}

type caller struct {
	streams map[uint64]chan Event

	scope string

	nextOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	players map[Holder]time.Time

	outstanding string
	misses      int

	grants []time.Time

	owedBox Kind

	nextQuizAt      time.Time
	outstandingQuiz string
	quizMisses      int
	quizGrants      []time.Time

	owedQuiz Kind
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
	Offered func()

	Lapsed func(scope string)

	Caught func(scope string, after time.Duration)

	Foreign func(scope string)

	QuizOffered func()

	QuizLapsed func(scope string)

	QuizAnswered func(scope string, correct bool, after time.Duration)
}

type Pace struct {
	MinInterval time.Duration
	MaxInterval time.Duration

	MaxChargesPerHour int
}

type Registry struct {
	config   Config
	rules    InForce
	clock    cptime.Clock
	report   Report
	holdings Holdings
	bank     Questions

	mu            sync.Mutex
	followedBoxes Pace
	followedQuiz  QuizRules
	callers       map[Entrant]*caller
	offers        map[string]*pending

	spent map[string]spentOffer

	quizOffers map[string]*pendingQuiz

	newcomers map[Holder]*newcomer

	nextID uint64
}

type pending struct {
	entrant   Entrant
	scope     string
	kind      Kind
	offeredAt time.Time
	expiresAt time.Time
}

type spentOffer struct {
	entrant Entrant
	until   time.Time
}

const rememberSpent = 10 * time.Minute

const eventBuffer = 32

type Holdings interface {
	Held(ctx context.Context, holder Holder) (Held, error)
}

func New(config Config, clock cptime.Clock, holdings Holdings, rules InForce, bank Questions) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	followed := rules.Bonuses()

	return &Registry{
		config:        config.withDefaults(),
		rules:         rules,
		clock:         clock,
		holdings:      holdings,
		bank:          bank,
		followedBoxes: followed.Boxes,
		followedQuiz:  followed.Quiz,
		callers:       make(map[Entrant]*caller),
		offers:        make(map[string]*pending),
		spent:         make(map[string]spentOffer),
		quizOffers:    make(map[string]*pendingQuiz),
		newcomers:     make(map[Holder]*newcomer),
	}
}

func (r *Registry) Observe(report Report) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report = report
}

// follow brings in a box or a quiz due later than the longest wait of a new pace, and a caller's next quiz is a whole
// wait away from when quizzes are switched on.
func (r *Registry) follow(rules Rules, now time.Time) {
	if rules.Boxes != r.followedBoxes {
		for _, entry := range r.callers {
			if entry.nextOfferAt.After(now.Add(rules.Boxes.MaxInterval)) {
				entry.nextOfferAt = now.Add(boxWait(rules))
			}
		}
		r.followedBoxes = rules.Boxes
	}

	if rules.Quiz != r.followedQuiz {
		switchedOn := rules.Quiz.Enabled && !r.followedQuiz.Enabled
		for _, entry := range r.callers {
			if switchedOn || entry.nextQuizAt.After(now.Add(rules.Quiz.MaxInterval)) {
				entry.nextQuizAt = now.Add(quizWait(rules))
			}
		}
		r.followedQuiz = rules.Quiz
	}
}

func (r *Registry) counted(hook func()) {
	if hook != nil {
		hook()
	}
}

func (r *Registry) Attend(entrant Entrant) (<-chan Event, func()) {
	now := r.clock.Now()
	rules := r.rules.Bonuses()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(entrant, rules, now)

	r.nextID++
	id := r.nextID

	events := make(chan Event, eventBuffer)
	entry.streams[id] = events
	entry.lastSeen = now

	return events, func() { r.leave(entrant, id) }
}

func (r *Registry) caller(entrant Entrant, rules Rules, now time.Time) *caller {
	entry, ok := r.callers[entrant]
	if ok {
		return entry
	}

	entry = &caller{
		streams:     make(map[uint64]chan Event),
		players:     make(map[Holder]time.Time),
		nextOfferAt: now.Add(boxWait(rules)),
		nextQuizAt:  now.Add(quizWait(rules)),
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

func (r *Registry) Clicked(click Click) {
	now := r.clock.Now()
	rules := r.rules.Bonuses()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(click.Entrant, rules, now)
	entry.scope = click.Scope
	entry.lastClickAt = now
	if click.Holder == NoHolder {
		return
	}

	entry.players[click.Holder] = now
	r.welcome(entry, click, rules, now)
}

func (r *Registry) Claim(token string, entrant Entrant, scope string) (Reward, bool) {
	now := r.clock.Now()
	rules := r.rules.Bonuses()

	r.mu.Lock()
	defer r.mu.Unlock()

	offer, ok := r.offers[token]
	if !ok || offer.entrant != entrant || !now.Before(offer.expiresAt) {
		r.refuse(token, entrant, scope)
		return Reward{}, false
	}

	delete(r.offers, token)
	r.spent[token] = spentOffer{entrant: entrant, until: now.Add(rememberSpent)}

	if r.report.Caught != nil {
		r.report.Caught(scope, now.Sub(offer.offeredAt))
	}

	if entry, known := r.callers[entrant]; known {
		entry.outstanding = ""
		entry.misses = 0
		entry.grants = append(entry.grants, now)

		entry.nextOfferAt = now.Add(boxWait(rules))
	}

	return Reward{Kind: offer.kind, Amount: amountOf(offer.kind, rules)}, true
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

func (r *Registry) Publish(taken Taken) {
	r.broadcast(Event{Taken: &taken})
}

func (r *Registry) PublishEnclosed(entrant Entrant, enclosed Enclosed) {
	r.mu.Lock()
	defer r.mu.Unlock()

	theirs := enclosed
	theirs.Yours = false

	for other, entry := range r.callers {
		if other == entrant {
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
			r.sweep(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Registry) sweep(ctx context.Context) {
	now := r.clock.Now()
	rules := r.rules.Bonuses()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.follow(rules, now)
	r.collectMisses(rules, now)
	r.forgetSpent(now)
	r.sweepQuizzes(ctx, rules, now)
	r.forgetStale(now)
	r.forgetNewcomers(now)

	for entrant, entry := range r.callers {
		if !r.due(entry, rules, now) {
			continue
		}

		kinds := drawable(entry.owedBox, r.offerable(ctx, entry, rules, now))
		entry.owedBox = ""
		if kinds.Empty() {
			entry.nextOfferAt = now.Add(boxWait(rules))
			continue
		}

		r.offer(entrant, entry, rules, now, kinds)
	}
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

func (r *Registry) offerable(ctx context.Context, entry *caller, rules Rules, now time.Time) *cpcolls.Set[Kind] {
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))

	held := cpcolls.NewSet[Kind]()
	for holder, clicked := range entry.players {
		if now.Sub(clicked) > rules.ActiveWithin {
			delete(entry.players, holder)
			continue
		}
		hand, err := r.holdings.Held(ctx, holder)
		if err != nil {
			// Offered nothing: the slot is lost, as an away player's.
			return kinds
		}
		held.Add(hand.Full(rules)...)
	}

	if len(entry.players) == 0 || r.grantedWithinTheHour(entry, now) >= rules.Boxes.MaxChargesPerHour {
		return kinds
	}

	for _, kind := range Kinds {
		if rules.Kinds[kind] > 0 && !held.Contains(kind) {
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

func (r *Registry) offer(entrant Entrant, entry *caller, rules Rules, now time.Time, kinds *cpcolls.Set[Kind]) {
	token, err := newToken()
	if err != nil {
		return
	}

	kind := drawKind(kinds, rules.Kinds)
	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		Kind:      kind,
		ExpiresAt: now.Add(rules.OfferTTL),
	}

	r.offers[token] = &pending{
		entrant:   entrant,
		scope:     entry.scope,
		kind:      offer.Kind,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstanding = token
	entry.nextOfferAt = offer.ExpiresAt.Add(boxWait(rules))

	entry.send(Event{Offer: &offer})
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

		if entry.outstandingQuiz != "" {
			delete(r.quizOffers, entry.outstandingQuiz)
		}

		delete(r.callers, entrant)
	}
}

func boxWait(rules Rules) time.Duration {
	return drawWindow(rules.Boxes.MinInterval, rules.Boxes.MaxInterval)
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

func drawable(owed Kind, kinds *cpcolls.Set[Kind]) *cpcolls.Set[Kind] {
	if owed == "" {
		return kinds
	}

	if kinds.Contains(owed) {
		return cpcolls.NewSet(owed)
	}

	return cpcolls.NewSet[Kind]()
}

func drawKind(kinds *cpcolls.Set[Kind], weights map[Kind]float64) Kind {
	total := 0.0
	var last Kind
	for _, kind := range Kinds {
		if kinds.Contains(kind) {
			total += weights[kind]
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
		weight := weights[kind]
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

func amountOf(kind Kind, rules Rules) int {
	least, most := 1, 1
	switch kind {
	case KindSpreadClicks:
		most = rules.Spread.MaxPerBox
	case KindEncloseClicks:
		most = rules.Enclose.MaxPerBox
	case KindShields:
		least, most = rules.Shield.MinPerBox, rules.Shield.MaxPerBox
	case KindRefill, KindBomb:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(most-least+1)))
	if err != nil {
		return least
	}

	return least + int(n.Int64())
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
