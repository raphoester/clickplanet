package bonuses

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
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

// Event is what the offers send down a caller's streams: a box or a quiz for it, and every catch for all.
type Event struct {
	Offer *Offer
	Quiz  *QuizOffer
	Taken *Taken
}

type Reward struct {
	Kind Kind

	Amount int
}

// Click is a click planet took, struck a shield with or found its tile already wearing its flag, as planet's ClickTried
// tells it: what it says of its caller, when, and whether it left the bank empty.
type Click struct {
	Caller Caller

	// At is when it was tried. Zero is when it is heard.
	At time.Time

	// Born is when the account was made, read from its id.
	Born time.Time

	// Dry is a click that left the bank empty: fewer than one click.
	Dry bool
}

type caller struct {
	streams map[uint64]chan Event

	scope string

	nextOfferAt time.Time
	lastOfferAt time.Time
	lastSeen    time.Time
	lastClickAt time.Time

	players map[Account]time.Time

	outstanding string
	misses      int

	grants []time.Time

	owedBox Kind

	nextQuizAt      time.Time
	lastQuizAt      time.Time
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
	config Config
	rules  func() Rules
	clock  cptime.Clock
	report Report
	room   Room
	bank   Questions

	mu            sync.Mutex
	followedBoxes Pace
	followedQuiz  QuizRules
	callers       map[Entrant]*caller
	offers        map[string]*pending

	spent map[string]spentOffer

	quizOffers map[string]*pendingQuiz

	newcomers map[Account]*newcomer

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

func New(config Config, clock cptime.Clock, room Room, rules func() Rules, bank Questions) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	followed := rules()

	return &Registry{
		config:        config.withDefaults(),
		rules:         rules,
		clock:         clock,
		room:          room,
		bank:          bank,
		followedBoxes: followed.Boxes,
		followedQuiz:  followed.Quiz,
		callers:       make(map[Entrant]*caller),
		offers:        make(map[string]*pending),
		spent:         make(map[string]spentOffer),
		quizOffers:    make(map[string]*pendingQuiz),
		newcomers:     make(map[Account]*newcomer),
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
	rules := r.rules()

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
		players:     make(map[Account]time.Time),
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

func (r *Registry) Claim(token string, entrant Entrant, scope string) (Reward, bool) {
	now := r.clock.Now()
	rules := r.rules()

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

// sweep decides who is due under the lock, asks the wallet for their room outside it, then offers under it again to
// whoever is due still: the room is a call out of the offers, and nothing else waits on the registry while it runs.
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
	box      bool
	quiz     bool
	accounts []Account

	full map[Account][]Kind
	err  error
}

// dueCallers are the callers due a quiz or a box, once the registry is tidied under the same lock.
func (r *Registry) dueCallers(rules Rules, now time.Time) []*dueOne {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tidy(rules, now)

	var due []*dueOne
	for entrant, entry := range r.callers {
		quiz := rules.Quiz.Enabled && r.quizDue(entry, rules, now)
		box := r.due(entry, rules, now)
		if quiz || box {
			due = append(due, &dueOne{entrant: entrant, box: box, quiz: quiz, accounts: entry.playing(rules, now)})
		}
	}

	return due
}

func (r *Registry) tidy(rules Rules, now time.Time) {
	r.follow(rules, now)
	r.collectMisses(rules, now)
	r.forgetSpent(now)
	r.collectStaleQuizzes(rules, now)
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
			one.full, one.err = r.room.Full(ctx, one.accounts)
		}
	}
}

// offerDue offers the quizzes first, then the boxes, so a box due beside a quiz just offered waits for it.
func (r *Registry) offerDue(due []*dueOne, rules Rules, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, one := range due {
		entry, ok := r.callers[one.entrant]
		if !one.quiz || !ok || !entry.playedBy(one.accounts, rules, now) || !r.quizDue(entry, rules, now) {
			continue
		}

		kinds := drawable(entry.owedQuiz, r.offerable(entry, one, rules, now))
		entry.owedQuiz = ""
		if kinds.Empty() {
			entry.nextQuizAt = now.Add(quizWait(rules))
			continue
		}

		r.offerQuiz(one.entrant, entry, rules, now, drawKind(kinds, rules.Kinds))
	}

	for _, one := range due {
		entry, ok := r.callers[one.entrant]
		if !one.box || !ok || !entry.playedBy(one.accounts, rules, now) || !r.due(entry, rules, now) {
			continue
		}

		kinds := drawable(entry.owedBox, r.offerable(entry, one, rules, now))
		entry.owedBox = ""
		if kinds.Empty() {
			entry.nextOfferAt = now.Add(boxWait(rules))
			continue
		}

		r.offer(one.entrant, entry, rules, now, kinds)
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

	return !near(entry.outstandingQuiz, entry.lastQuizAt, entry.owedBox, rules, now)
}

// offerable leaves out every kind the wallet said an account playing behind the caller has no room for.
func (r *Registry) offerable(entry *caller, one *dueOne, rules Rules, now time.Time) *cpcolls.Set[Kind] {
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))

	if one.err != nil || rules.Frozen {
		// Offered nothing: the slot is lost, as an away player's.
		return kinds
	}

	if len(one.accounts) == 0 || r.grantedWithinTheHour(entry, now) >= rules.Boxes.MaxChargesPerHour {
		return kinds
	}

	full := cpcolls.NewSet[Kind]()
	for _, account := range one.accounts {
		full.Add(one.full[account]...)
	}

	for _, kind := range Kinds {
		if rules.Offered(kind) && !full.Contains(kind) {
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
	entry.lastOfferAt = now
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

// A box and a quiz keep this far apart: half the shorter of their shortest waits, so one falls between two of the other
// rather than beside one.
func spacing(rules Rules) time.Duration {
	return min(rules.Boxes.MinInterval, rules.Quiz.MinInterval) / 2
}

// near says the other of the two, out still or offered at offeredAt, is too close for this one to be offered now. It
// is held, not redrawn, so it comes the moment the other is far enough. A welcome owed has its own moment, so it waits
// only for the other to be gone.
func near(out string, offeredAt time.Time, owed Kind, rules Rules, now time.Time) bool {
	if out != "" {
		return true
	}

	return owed == "" && now.Sub(offeredAt) < spacing(rules)
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
