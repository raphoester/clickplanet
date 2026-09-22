package bonuses

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

// The quiz half of the schedule: a question put in front of one caller, for a charge.
//
// It lives here rather than in its own registry because a caller is one thing — the streams it has
// open, when it last clicked, which accounts are playing behind it — and that bookkeeping should
// not exist twice. What is genuinely the quiz's own is all in this file: its own clock, its own
// hourly budget, and its own rule for when an offer may go out.
//
// It is a **second way to earn a charge**, not a differently shaped box. The two schedules do not
// know about each other: a caller can be holding an unopened quiz and a flying box at the same
// time, and each pays into its own MaxChargesPerHour.
//
// What it does share with a box, and must: the reward. A quiz's kind is drawn at offer time from
// exactly the kinds a box would be drawn from, so a player already holding a bomb is never asked a
// question for another one.

// QuizOffer is the banner, and the token that opens it. No question and no choices: those are read
// with Open, which is what starts the clock.
type QuizOffer struct {
	Token string

	// When the banner is gone. The invitation lapsing, not the answer clock.
	ExpiresAt time.Time

	// The country the question is about, for the banner's flag. Empty for a question about nowhere.
	Subject string
}

// Asked is the question as it was put, and how long is left to answer it.
type Asked struct {
	Question string

	// Exactly quizzes.Choices, in the order they were sent. Which one is right is not in here.
	Options []string

	Deadline time.Time

	// The whole window, so the client draws the countdown against what it was given rather than
	// against what is left after a slow round trip.
	Window time.Duration
}

// Answered is what a quiz was worth. Reward is empty unless Correct.
type Answered struct {
	Correct bool

	// Which option was the right one, whatever was pressed.
	CorrectChoice int

	Reward Reward

	// The country the question was about, for the broadcast.
	Subject string
}

// Questions is the bank a quiz is drawn from. The registry never sees the answer text: it holds the
// round and compares an index, so "which one is right" has nowhere to leak to.
type Questions interface {
	Draw() quizzes.Round
}

// pendingQuiz is one offer, from the banner going out to the answer landing.
type pendingQuiz struct {
	scope     string
	round     quizzes.Round
	kind      Kind
	offeredAt time.Time

	// When the banner stops being clickable.
	expiresAt time.Time

	// Zero until the player opens it. Stamping it once is what makes a reload cost nothing and buy
	// nothing: the second open answers the same deadline as the first.
	deadline time.Time
}

func (p *pendingQuiz) opened() bool { return !p.deadline.IsZero() }

// Quizzing switches the quizzes on and hands the registry the bank to draw from. Left uncalled, no
// quiz is ever offered and the boxes fly exactly as before — which is what a deploy with the
// feature off looks like, and what every test that is not about quizzes gets.
func (r *Registry) Quizzing(config quizzes.Config, bank Questions) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.quizConfig = config.Settings()
	r.bank = bank
}

func (r *Registry) quizzing() bool { return r.bank != nil && r.quizConfig.Enabled }

// OpenQuiz reads the question and starts its clock, or says there is no such quiz to open.
//
// Opening twice is not an error and is not a second chance: the deadline is stamped on the first
// open and answered again on every one after, so a reload shows the same question with less time on
// it. That is the honest reading — the player did have those seconds.
func (r *Registry) OpenQuiz(token string, scope string) (Asked, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.quizOffers[token]
	if !ok || quiz.scope != scope || !now.Before(quiz.expiresAt) {
		return Asked{}, false
	}

	if !quiz.opened() {
		quiz.deadline = now.Add(r.quizConfig.AnswerWindow)
	}

	return Asked{
		Question: quiz.round.Question.Text,
		Options:  quiz.round.Options,
		Deadline: quiz.deadline,
		Window:   r.quizConfig.AnswerWindow,
	}, true
}

// AnswerQuiz spends the token and says what the answer was worth. It fails only for a quiz that is
// not this caller's to answer: unknown, already answered, or never opened. A wrong answer and a
// late one both succeed — they are worth nothing, which is not the same as not happening.
func (r *Registry) AnswerQuiz(token string, scope string, choice int) (Answered, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.quizOffers[token]
	if !ok || quiz.scope != scope {
		return Answered{}, false
	}

	// An answer to a question nobody read is a client that skipped OpenQuiz, which is a client
	// guessing at three choices it was never sent.
	if !quiz.opened() {
		return Answered{}, false
	}

	delete(r.quizOffers, token)

	entry, known := r.callers[scope]
	if known && entry.outstandingQuiz == token {
		entry.outstandingQuiz = ""
		entry.nextQuizAt = now.Add(r.quizWindow())
	}

	answered := Answered{
		CorrectChoice: quiz.round.Correct,
		Subject:       quiz.round.Question.Subject,
		// In time, and right. The deadline is the server's own stamp, so a client that holds its
		// countdown open cannot spend longer than it was given.
		Correct: !now.After(quiz.deadline) && quiz.round.Correctly(choice),
	}

	if r.report.QuizAnswered != nil {
		r.report.QuizAnswered(scope, answered.Correct, now.Sub(quiz.offeredAt))
	}

	if !answered.Correct {
		return answered, true
	}

	if known {
		entry.quizGrants = append(entry.quizGrants, now)
	}
	answered.Reward = Reward{Kind: quiz.kind, Amount: r.amountOf(quiz.kind)}

	return answered, true
}

// sweepQuizzes retires the banners nobody opened and puts out the ones that are due. Called from
// the one sweep, with the lock held.
func (r *Registry) sweepQuizzes(now time.Time) {
	if !r.quizzing() {
		return
	}

	r.collectStaleQuizzes(now)

	for scope, entry := range r.callers {
		if !r.quizDue(entry, now) {
			continue
		}

		kinds := r.offerable(entry, now)
		if kinds.Empty() {
			// Nothing a right answer could be worth, so there is nothing to ask for. The slot is
			// lost rather than banked, as a box's is.
			entry.nextQuizAt = now.Add(r.quizWindow())
			continue
		}

		r.offerQuiz(scope, entry, now, r.drawKind(kinds))
	}
}

// quizDue is the box rule with the quiz's own clock and the quiz's own budget: a caller who is
// watching, who is playing, who is not already holding one, and who has not won its hour's worth.
func (r *Registry) quizDue(entry *caller, now time.Time) bool {
	if !entry.watching() || entry.outstandingQuiz != "" || now.Before(entry.nextQuizAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > r.config.ActiveWithin {
		entry.nextQuizAt = now.Add(r.quizWindow())
		return false
	}

	if r.quizzesWonWithinTheHour(entry, now) >= r.quizConfig.MaxChargesPerHour {
		entry.nextQuizAt = now.Add(r.quizWindow())
		return false
	}

	return true
}

func (r *Registry) offerQuiz(scope string, entry *caller, now time.Time, kind Kind) {
	token, err := newToken()
	if err != nil {
		return
	}

	round := r.bank.Draw()
	offer := QuizOffer{
		Token:     token,
		ExpiresAt: now.Add(r.quizConfig.OfferTTL),
		Subject:   round.Question.Subject,
	}

	r.quizOffers[token] = &pendingQuiz{
		scope:     scope,
		round:     round,
		kind:      kind,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstandingQuiz = token
	// A banner opened at the last second still has its own window to answer in, so the schedule is
	// parked past the latest an answer could land. Whatever actually happens to the quiz — answered,
	// or swept — moves it back to a window from then.
	entry.nextQuizAt = offer.ExpiresAt.Add(r.quizConfig.AnswerWindow).Add(r.quizWindow())

	entry.send(Event{Quiz: &offer})
	r.counted(r.report.QuizOffered)
}

// collectStaleQuizzes drops what can no longer be answered: a banner nobody clicked, and a question
// opened and left. Both free the caller's slot.
func (r *Registry) collectStaleQuizzes(now time.Time) {
	for token, quiz := range r.quizOffers {
		gone := now.After(quiz.expiresAt)
		if quiz.opened() {
			gone = now.After(quiz.deadline)
		}
		if !gone {
			continue
		}

		delete(r.quizOffers, token)

		entry, ok := r.callers[quiz.scope]
		if !ok || entry.outstandingQuiz != token {
			continue
		}

		entry.outstandingQuiz = ""
		// A window from when it went, not from when it was offered: the slot is lost rather than
		// banked, and a player who ignores a banner waits the same as one who answered.
		entry.nextQuizAt = now.Add(r.quizWindow())

		if r.report.QuizLapsed != nil {
			r.report.QuizLapsed(quiz.scope)
		}
	}
}

// quizzesWonWithinTheHour is how many charges quizzes granted this caller in the last hour. It
// forgets the older ones on the way.
func (r *Registry) quizzesWonWithinTheHour(entry *caller, now time.Time) int {
	since := now.Add(-time.Hour)

	kept := entry.quizGrants[:0]
	for _, at := range entry.quizGrants {
		if at.After(since) {
			kept = append(kept, at)
		}
	}
	entry.quizGrants = kept

	return len(kept)
}

func (r *Registry) quizWindow() time.Duration {
	return drawWindow(r.quizConfig.MinInterval, r.quizConfig.MaxInterval)
}
