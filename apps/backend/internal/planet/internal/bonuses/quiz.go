package bonuses

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

// Must say nothing about the question: even its subject's flag can be the answer.
type QuizOffer struct {
	Token string

	ExpiresAt time.Time
}

type Asked struct {
	Question string

	Options []string

	Deadline time.Time

	Window time.Duration
}

type Answered struct {
	Correct bool

	CorrectChoice int

	Reward Reward

	Subject string
}

type Questions interface {
	Draw() quizzes.Round
}

type pendingQuiz struct {
	scope     string
	round     quizzes.Round
	kind      Kind
	offeredAt time.Time

	expiresAt time.Time

	deadline time.Time
}

func (p *pendingQuiz) opened() bool { return !p.deadline.IsZero() }

func (r *Registry) Quizzing(config quizzes.Config, bank Questions) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.quizConfig = config.Settings()
	r.bank = bank
}

func (r *Registry) quizzing() bool { return r.bank != nil && r.quizConfig.Enabled }

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

func (r *Registry) AnswerQuiz(token string, scope string, choice int) (Answered, bool) {
	now := r.clock.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.quizOffers[token]
	if !ok || quiz.scope != scope {
		return Answered{}, false
	}

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
		Correct:       !now.After(quiz.deadline) && quiz.round.Correctly(choice),
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

// offerQuizzes runs in the sweep's second hold of the lock, for the callers whose flags it read.
func (r *Registry) offerQuizzes(
	now time.Time, due map[string][]clicks.AllegianceKey, flags map[clicks.AllegianceKey]clicks.Allegiance,
) {
	if !r.quizzing() {
		return
	}

	for scope := range due {
		entry, known := r.callers[scope]
		if !known || !r.quizDue(entry, now) {
			continue
		}

		r.forgetIdlePlayers(entry, now)
		band, read := r.band(flags, scope, entry)
		if !read {
			continue
		}

		kinds := r.offerable(entry, now, band)
		if kinds.Empty() {
			entry.nextQuizAt = now.Add(r.quizWindow())
			continue
		}

		r.offerQuiz(scope, entry, now, band, drawKind(band, kinds))
	}
}

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

func (r *Registry) offerQuiz(scope string, entry *caller, now time.Time, band KindBand, kind Kind) {
	token, err := newToken()
	if err != nil {
		return
	}

	round := r.bank.Draw()
	offer := QuizOffer{Token: token, ExpiresAt: now.Add(r.quizConfig.OfferTTL)}

	r.quizOffers[token] = &pendingQuiz{
		scope:     scope,
		round:     round,
		kind:      kind,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstandingQuiz = token
	entry.nextQuizAt = offer.ExpiresAt.Add(r.quizConfig.AnswerWindow).Add(r.quizWindow())

	entry.send(Event{Quiz: &offer})
	if r.report.QuizOffered != nil {
		r.report.QuizOffered(kind, band.Share)
	}
}

func (r *Registry) collectStaleQuizzes(now time.Time) {
	if !r.quizzing() {
		return
	}

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
		entry.nextQuizAt = now.Add(r.quizWindow())

		if r.report.QuizLapsed != nil {
			r.report.QuizLapsed(quiz.scope)
		}
	}
}

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
