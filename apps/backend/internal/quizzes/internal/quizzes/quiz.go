package quizzes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Offer must say nothing about the question: even its subject's flag can be the answer.
type Offer struct {
	Token string

	ExpiresAt time.Time
}

// Won is a quiz answered right, told to everyone: once it is spent, its subject gives nothing away.
type Won struct {
	CountryID string

	Subject string
}

type Event struct {
	Offer *Offer
	Won   *Won
}

type Asked struct {
	Question string

	Options []string

	Deadline time.Time

	Window time.Duration
}

type Answered struct {
	Outcome ratings.Outcome

	CorrectChoice int

	Subject string

	// What the ratings count of it besides its outcome: the question, and how long after the opening it landed.
	Question ratings.QuestionID
	After    time.Duration

	// Late is an answer to a question opened and lapsed a moment before, which the page's timeout sent over a slow
	// network or a clock behind the server's. To the player it is a quiz gone, as before; the ratings count it wrong.
	Late bool

	// Welcome is the quiz asked early to a new account whose clicks ran out.
	Welcome bool
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

// An answer lands at most this long past its question's deadline to be counted, late: the page's timeout reaches the
// server a trip after the deadline, and after the sweep that collected it on a slow network or a clock behind ours.
const lateAnswers = 10 * time.Second

type Questions interface {
	Draw(lean float64, fit bank.Fit) bank.Round
}

// Every hook runs with the registry locked, so none may call back into it.
type Report struct {
	Offered func()

	Lapsed func(scope string)

	Answered func(scope string, correct bool, after time.Duration)
}

type caller struct {
	streams map[uint64]chan Event

	scope string

	lastSeen    time.Time
	lastClickAt time.Time

	nextAt      time.Time
	outstanding string
	misses      int
	won         []time.Time

	// owed is the welcome quiz, due at once to a newcomer whose clicks ran out.
	owed bool
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

type pending struct {
	entrant   Entrant
	scope     string
	round     bank.Round
	welcome   bool
	offeredAt time.Time

	expiresAt time.Time

	openedAt time.Time
	deadline time.Time
}

func (p *pending) opened() bool { return !p.deadline.IsZero() }

type Registry struct {
	config Config
	rules  func() Rules
	clock  cptime.Clock
	report Report
	bank   Questions

	mu       sync.Mutex
	followed pace
	callers  map[Entrant]*caller
	offers   map[string]*pending

	// Each question opened and lapsed, until lateAnswers past its deadline: an answer that lands then is counted, late.
	lapsed map[string]*pending

	// Each newcomer asked its welcome quiz, until it is no longer new.
	newcomers map[Account]time.Time

	nextID uint64
}

const eventBuffer = 32

func New(config Config, clock cptime.Clock, rules func() Rules, questions Questions) *Registry {
	return &Registry{
		config:    config.withDefaults(),
		rules:     rules,
		clock:     clock,
		bank:      questions,
		followed:  rules().pace(),
		callers:   make(map[Entrant]*caller),
		offers:    make(map[string]*pending),
		lapsed:    make(map[string]*pending),
		newcomers: make(map[Account]time.Time),
	}
}

func (r *Registry) Observe(report Report) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.report = report
}

// follow brings in a quiz due later than the longest wait of a new pace, and a caller's next quiz is a whole wait away
// from when quizzes are switched on.
func (r *Registry) follow(rules Rules, now time.Time) {
	followed := rules.pace()
	if followed == r.followed {
		return
	}

	switchedOn := followed.enabled && !r.followed.enabled
	for _, entry := range r.callers {
		if switchedOn || entry.nextAt.After(now.Add(rules.MaxInterval)) {
			entry.nextAt = now.Add(wait(rules))
		}
	}
	r.followed = followed
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
		streams:  make(map[uint64]chan Event),
		nextAt:   now.Add(wait(rules)),
		lastSeen: now,
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

// Clicked hears a click a moment after it was tried. A click with no account, or one tried longer ago than
// activeWithin, heard late after a boot, is no sign of a player.
func (r *Registry) Clicked(click Click) {
	now := r.clock.Now()
	rules := r.rules()

	at := click.At
	if at.IsZero() || at.After(now) {
		at = now
	}
	if click.Caller.Account == NoAccount || now.Sub(at) > rules.ActiveWithin {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.caller(EntrantOf(click.Caller), rules, now)
	entry.scope = click.Caller.Scope
	if at.After(entry.lastClickAt) {
		entry.lastClickAt = at
	}
	r.welcome(entry, click, rules, now)
}

func (r *Registry) OpenQuiz(token string, entrant Entrant) (Asked, bool) {
	now := r.clock.Now()
	window := r.rules().AnswerWindow

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.offers[token]
	if !ok || quiz.entrant != entrant || (!quiz.opened() && !now.Before(quiz.expiresAt)) {
		return Asked{}, false
	}

	if !quiz.opened() {
		quiz.openedAt = now
		quiz.deadline = now.Add(window)
	}

	return Asked{
		Question: quiz.round.Question.Text,
		Options:  quiz.round.Options,
		Deadline: quiz.deadline,
		Window:   window,
	}, true
}

func (r *Registry) AnswerQuiz(token string, entrant Entrant, choice int) (Answered, bool) {
	now := r.clock.Now()
	rules := r.rules()

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.offers[token]
	if !ok {
		return r.answerLate(token, entrant, choice, now)
	}
	if quiz.entrant != entrant || !quiz.opened() {
		return Answered{}, false
	}

	delete(r.offers, token)

	answered := quiz.answered(choice, now)
	if r.report.Answered != nil {
		r.report.Answered(quiz.scope, answered.Outcome == ratings.Right, now.Sub(quiz.offeredAt))
	}

	if entry, known := r.callers[entrant]; known && entry.outstanding == token {
		entry.outstanding = ""
		entry.misses = 0
		entry.nextAt = now.Add(wait(rules))
		if answered.Outcome == ratings.Right {
			entry.won = append(entry.won, now)
		}
	}

	return answered, true
}

// answerLate counts an answer to a question opened and lapsed less than lateAnswers ago, once: the quiz is gone, and the
// answer is wrong and only counted.
func (r *Registry) answerLate(token string, entrant Entrant, choice int, now time.Time) (Answered, bool) {
	quiz, ok := r.lapsed[token]
	if !ok || quiz.entrant != entrant || now.After(quiz.deadline.Add(lateAnswers)) {
		return Answered{}, false
	}

	delete(r.lapsed, token)

	answered := quiz.answered(choice, now)
	answered.Late = true

	return answered, true
}

func (p *pending) answered(choice int, now time.Time) Answered {
	return Answered{
		Outcome:       p.outcome(choice, now),
		CorrectChoice: p.round.Correct,
		Subject:       p.round.Question.Subject,
		Question:      p.round.Question.ID,
		After:         now.Sub(p.openedAt),
		Welcome:       p.welcome,
	}
}

// The page's own timeout is a choice past the three it was shown.
func (p *pending) outcome(choice int, now time.Time) ratings.Outcome {
	if choice < 0 || choice >= len(p.round.Options) {
		return ratings.TimedOut
	}
	if now.After(p.deadline) || !p.round.Correctly(choice) {
		return ratings.Wrong
	}
	return ratings.Right
}

func (r *Registry) Publish(won Won) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.callers {
		entry.send(Event{Won: &won})
	}
}

func (r *Registry) Name() string { return "quizzes" }

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

// sweep tidies the registry and offers a quiz to each caller due. A frozen set offers nothing: each slot that comes due
// then is lost, as an away player's, and the next wait is drawn.
func (r *Registry) sweep() {
	now := r.clock.Now()
	rules := r.rules()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.follow(rules, now)
	r.collectLapsed(rules, now)
	r.forgetStale(now)
	r.forgetNewcomers(now)

	if !rules.Enabled {
		return
	}
	for entrant, entry := range r.callers {
		if !r.due(entry, rules, now) {
			continue
		}
		if rules.Frozen {
			entry.owed = false
			entry.nextAt = now.Add(wait(rules))
			continue
		}
		r.offer(entrant, entry, rules, now)
	}
}

func (r *Registry) due(entry *caller, rules Rules, now time.Time) bool {
	if !entry.watching() || entry.outstanding != "" || now.Before(entry.nextAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > rules.ActiveWithin || r.wonWithinTheHour(entry, now) >= rules.MaxRightAnswersPerHour {
		entry.nextAt = now.Add(wait(rules))
		return false
	}

	return true
}

func (r *Registry) offer(entrant Entrant, entry *caller, rules Rules, now time.Time) {
	token, err := newToken()
	if err != nil {
		return
	}

	offer := Offer{Token: token, ExpiresAt: now.Add(rules.OfferTTL)}
	r.offers[token] = &pending{
		entrant:   entrant,
		scope:     entry.scope,
		round:     r.bank.Draw(rules.LeaderBias, rules.Fit),
		welcome:   entry.owed,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.owed = false
	entry.outstanding = token
	entry.nextAt = offer.ExpiresAt.Add(rules.AnswerWindow).Add(wait(rules))

	entry.send(Event{Offer: &offer})
	if r.report.Offered != nil {
		r.report.Offered()
	}
}

func (r *Registry) collectLapsed(rules Rules, now time.Time) {
	for token, quiz := range r.lapsed {
		if now.After(quiz.deadline.Add(lateAnswers)) {
			delete(r.lapsed, token)
		}
	}

	for token, quiz := range r.offers {
		gone := now.After(quiz.expiresAt)
		if quiz.opened() {
			gone = now.After(quiz.deadline)
		}
		if !gone {
			continue
		}

		delete(r.offers, token)
		if quiz.opened() {
			r.lapsed[token] = quiz
		}

		entry, ok := r.callers[quiz.entrant]
		if !ok || entry.outstanding != token {
			continue
		}

		entry.outstanding = ""
		entry.nextAt = now.Add(wait(rules))
		if entry.misses == 0 {
			entry.nextAt = now.Add(rules.MissRetry)
		}
		entry.misses++

		if r.report.Lapsed != nil {
			r.report.Lapsed(quiz.scope)
		}
	}
}

func (r *Registry) forgetStale(now time.Time) {
	for entrant, entry := range r.callers {
		if entry.watching() || now.Sub(entry.lastSeen) <= r.config.ForgetAfter {
			continue
		}

		if entry.outstanding != "" {
			delete(r.offers, entry.outstanding)
		}

		delete(r.callers, entrant)
	}
}

func (r *Registry) wonWithinTheHour(entry *caller, now time.Time) int {
	since := now.Add(-time.Hour)

	kept := entry.won[:0]
	for _, at := range entry.won {
		if at.After(since) {
			kept = append(kept, at)
		}
	}
	entry.won = kept

	return len(kept)
}

func wait(rules Rules) time.Duration {
	spread := rules.MaxInterval - rules.MinInterval
	if spread <= 0 {
		return rules.MinInterval
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(spread)))
	if err != nil {
		return rules.MinInterval
	}

	return rules.MinInterval + time.Duration(n.Int64())
}

func newToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to read random bytes for a quiz token: %w", err)
	}

	return hex.EncodeToString(raw), nil
}
