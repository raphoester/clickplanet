package bonuses

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
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

	scope string

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
	// Offered is a box's kind, and the share its band starts at.
	Offered func(kind Kind, band float64)

	Lapsed func(scope string)

	Caught func(scope string, after time.Duration)

	Foreign func(scope string)

	QuizOffered func(kind Kind, band float64)

	QuizLapsed func(scope string)

	QuizAnswered func(scope string, correct bool, after time.Duration)
}

type Registry struct {
	config   Config
	clock    cptime.Clock
	report   Report
	holdings Holdings
	shares   Shares
	flags    Flags
	bands    Bands
	charges  ChargesConfig

	quizConfig quizzes.Config
	bank       Questions

	mu      sync.Mutex
	callers map[Entrant]*caller
	offers  map[string]*pending

	spent map[string]spentOffer

	quizOffers map[string]*pendingQuiz

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
	Held(holder Holder) Held
}

// Shares is how much of the map a country holds, 0 to 1: the same count the toll reads.
type Shares interface {
	Share(country string) float64
}

// Flags is every tally of who takes tiles for which flag, by key, which the band is read from. A key with no
// tally is absent.
type Flags interface {
	Allegiances(ctx context.Context, keys ...clicks.AllegianceKey) (map[clicks.AllegianceKey]clicks.Allegiance, error)
}

// flagsTimeout bounds a sweep's read of the flags, so a slow database delays the boxes and nothing else.
const flagsTimeout = 2 * time.Second

func New(config Config, clock cptime.Clock, holdings Holdings, shares Shares, flags Flags) *Registry {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	config = config.withDefaults()

	return &Registry{
		config:     config,
		charges:    config.ChargesConfig(),
		bands:      config.Bands(),
		clock:      clock,
		holdings:   holdings,
		shares:     shares,
		flags:      flags,
		callers:    make(map[Entrant]*caller),
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

func (r *Registry) Attend(entrant Entrant) (<-chan Event, func()) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(entrant, now)

	r.nextID++
	id := r.nextID

	events := make(chan Event, eventBuffer)
	entry.streams[id] = events
	entry.lastSeen = now

	return events, func() { r.leave(entrant, id) }
}

func (r *Registry) caller(entrant Entrant, now time.Time) *caller {
	entry, ok := r.callers[entrant]
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

func (r *Registry) Clicked(entrant Entrant, scope string, holder Holder) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(entrant, now)
	entry.scope = scope
	entry.lastClickAt = now
	if holder != NoHolder {
		entry.players[holder] = now
	}
}

func (r *Registry) Claim(token string, entrant Entrant, scope string) (Reward, bool) {
	now := r.clock.Now()

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

		entry.nextOfferAt = now.Add(r.window())
	}

	return Reward{Kind: offer.kind, Amount: r.amountOf(offer.kind)}, true
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

// sweep reads the flags of every caller due a box or a quiz between two holds of the lock, never under it: the
// read waits on postgres, and every click takes the lock.
func (r *Registry) sweep(ctx context.Context) {
	now := r.clock.Now()

	due := r.tidy(now)
	if len(due) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, flagsTimeout)
	defer cancel()

	flags, err := r.readFlags(ctx, due)
	if err != nil {
		// Nobody is offered anything this sweep; the callers stay due and the next sweep reads again.
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.offerQuizzes(now, due, flags)

	for entrant := range due {
		entry, known := r.callers[entrant]
		if !known || !r.due(entry, now) {
			continue
		}

		r.forgetIdlePlayers(entry, now)
		band, read := r.band(flags, entry)
		if !read {
			continue
		}

		kinds := r.offerable(entry, now, band)
		if kinds.Empty() {
			entry.nextOfferAt = now.Add(r.window())
			continue
		}

		r.offer(entrant, entry, now, band, kinds)
	}
}

// tidy forgets what lapsed, and says which callers are due a box or a quiz, with the flags their band needs.
func (r *Registry) tidy(now time.Time) map[Entrant][]clicks.AllegianceKey {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.collectMisses(now)
	r.forgetSpent(now)
	r.collectStaleQuizzes(now)
	r.forgetStale(now)

	due := map[Entrant][]clicks.AllegianceKey{}
	for entrant, entry := range r.callers {
		if r.due(entry, now) || (r.quizzing() && r.quizDue(entry, now)) {
			r.forgetIdlePlayers(entry, now)
			due[entrant] = keysOf(entry)
		}
	}

	return due
}

// readFlags reads every tally the due callers need. Each key asked for is in the answer, a zero tally when it has
// no row, so a key missing from it is one this sweep did not read.
func (r *Registry) readFlags(
	ctx context.Context, due map[Entrant][]clicks.AllegianceKey,
) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	var keys []clicks.AllegianceKey
	for _, needed := range due {
		keys = append(keys, needed...)
	}

	found, err := r.flags.Allegiances(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags: %w", err)
	}

	flags := make(map[clicks.AllegianceKey]clicks.Allegiance, len(keys))
	for _, key := range keys {
		flags[key] = found[key]
	}

	return flags, nil
}

// keysOf is the flags a caller's band is read from: the scope it plays from, and each of its players'.
func keysOf(entry *caller) []clicks.AllegianceKey {
	keys := make([]clicks.AllegianceKey, 0, 1+len(entry.players))
	keys = append(keys, clicks.ScopeAllegianceKey(entry.scope))
	for holder := range entry.players {
		keys = append(keys, clicks.AccountAllegianceKey(string(holder)))
	}

	return keys
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

func (r *Registry) offerable(entry *caller, now time.Time, band KindBand) *cpcolls.Set[Kind] {
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))
	if len(entry.players) == 0 || r.grantedWithinTheHour(entry, now) >= r.config.MaxChargesPerHour {
		return kinds
	}

	held := cpcolls.NewSet[Kind]()
	for holder := range entry.players {
		held.Add(r.holdings.Held(holder).Full(r.charges)...)
	}

	for _, kind := range Kinds {
		if band.Kinds[kind] > 0 && !held.Contains(kind) {
			kinds.Add(kind)
		}
	}

	return kinds
}

// forgetIdlePlayers drops the players who have not clicked from the scope within ActiveWithin.
func (r *Registry) forgetIdlePlayers(entry *caller, now time.Time) {
	for holder, clicked := range entry.players {
		if now.Sub(clicked) > r.config.ActiveWithin {
			delete(entry.players, holder)
		}
	}
}

// band is the band of the biggest country among the scope's main flag and each playing account's: any of them can
// claim the box. False when a flag it needs is not in flags: a player who joined since they were read.
func (r *Registry) band(flags map[clicks.AllegianceKey]clicks.Allegiance, entry *caller) (KindBand, bool) {
	share := 0.0
	for _, key := range keysOf(entry) {
		tally, read := flags[key]
		if !read {
			return KindBand{}, false
		}
		share = max(share, r.shares.Share(tally.Flag()))
	}

	return r.bands.At(share), true
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

func (r *Registry) offer(entrant Entrant, entry *caller, now time.Time, band KindBand, kinds *cpcolls.Set[Kind]) {
	token, err := newToken()
	if err != nil {
		return
	}

	kind := drawKind(band, kinds)
	offer := Offer{
		Token:     token,
		Seed:      randomSeed(),
		Kind:      kind,
		ExpiresAt: now.Add(r.config.OfferTTL),
	}

	r.offers[token] = &pending{
		entrant:   entrant,
		scope:     entry.scope,
		kind:      offer.Kind,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstanding = token
	entry.nextOfferAt = offer.ExpiresAt.Add(r.window())

	entry.send(Event{Offer: &offer})
	if r.report.Offered != nil {
		r.report.Offered(kind, band.Share)
	}
}

func (r *Registry) collectMisses(now time.Time) {
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

func drawKind(band KindBand, kinds *cpcolls.Set[Kind]) Kind {
	total := 0.0
	var last Kind
	for _, kind := range Kinds {
		if kinds.Contains(kind) {
			total += band.Kinds[kind]
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
		weight := band.Kinds[kind]
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
