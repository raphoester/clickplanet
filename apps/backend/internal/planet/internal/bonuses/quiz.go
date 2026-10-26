package bonuses

import (
	"context"
	"time"

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
	Draw(lean float64) quizzes.Round
}

type pendingQuiz struct {
	entrant   Entrant
	scope     string
	round     quizzes.Round
	kind      Kind
	offeredAt time.Time

	expiresAt time.Time

	deadline time.Time
}

func (p *pendingQuiz) opened() bool { return !p.deadline.IsZero() }

func (r *Registry) OpenQuiz(token string, entrant Entrant) (Asked, bool) {
	now := r.clock.Now()
	window := r.rules().Quiz.AnswerWindow

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.quizOffers[token]
	if !ok || quiz.entrant != entrant || !now.Before(quiz.expiresAt) {
		return Asked{}, false
	}

	if !quiz.opened() {
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

	quiz, ok := r.quizOffers[token]
	if !ok || quiz.entrant != entrant {
		return Answered{}, false
	}

	if !quiz.opened() {
		return Answered{}, false
	}

	delete(r.quizOffers, token)

	entry, known := r.callers[entrant]
	if known && entry.outstandingQuiz == token {
		entry.outstandingQuiz = ""
		entry.quizMisses = 0
		entry.nextQuizAt = now.Add(quizWait(rules))
	}

	answered := Answered{
		CorrectChoice: quiz.round.Correct,
		Subject:       quiz.round.Question.Subject,
		Correct:       !now.After(quiz.deadline) && quiz.round.Correctly(choice),
	}

	if r.report.QuizAnswered != nil {
		r.report.QuizAnswered(quiz.scope, answered.Correct, now.Sub(quiz.offeredAt))
	}

	if !answered.Correct {
		return answered, true
	}

	if known {
		entry.quizGrants = append(entry.quizGrants, now)
	}
	answered.Reward = Reward{Kind: quiz.kind, Amount: amountOf(quiz.kind, rules)}

	return answered, true
}

// Switched off, no quiz is asked; one asked already can still be opened and answered until it lapses.
func (r *Registry) sweepQuizzes(ctx context.Context, rules Rules, now time.Time) {
	r.collectStaleQuizzes(rules, now)
	if !rules.Quiz.Enabled {
		return
	}

	for entrant, entry := range r.callers {
		if !r.quizDue(entry, rules, now) {
			continue
		}

		kinds := drawable(entry.owedQuiz, r.offerable(ctx, entry, rules, now))
		entry.owedQuiz = ""
		if kinds.Empty() {
			entry.nextQuizAt = now.Add(quizWait(rules))
			continue
		}

		r.offerQuiz(entrant, entry, rules, now, drawKind(kinds, rules.Kinds))
	}
}

func (r *Registry) quizDue(entry *caller, rules Rules, now time.Time) bool {
	if !entry.watching() || entry.outstandingQuiz != "" || now.Before(entry.nextQuizAt) {
		return false
	}

	if now.Sub(entry.lastClickAt) > rules.ActiveWithin {
		entry.nextQuizAt = now.Add(quizWait(rules))
		return false
	}

	if r.quizzesWonWithinTheHour(entry, now) >= rules.Quiz.MaxChargesPerHour {
		entry.nextQuizAt = now.Add(quizWait(rules))
		return false
	}

	return true
}

func (r *Registry) offerQuiz(entrant Entrant, entry *caller, rules Rules, now time.Time, kind Kind) {
	token, err := newToken()
	if err != nil {
		return
	}

	round := r.bank.Draw(rules.Quiz.LeaderBias)
	offer := QuizOffer{Token: token, ExpiresAt: now.Add(rules.Quiz.OfferTTL)}

	r.quizOffers[token] = &pendingQuiz{
		entrant:   entrant,
		scope:     entry.scope,
		round:     round,
		kind:      kind,
		offeredAt: now,
		expiresAt: offer.ExpiresAt,
	}

	entry.outstandingQuiz = token
	entry.nextQuizAt = offer.ExpiresAt.Add(rules.Quiz.AnswerWindow).Add(quizWait(rules))

	entry.send(Event{Quiz: &offer})
	r.counted(r.report.QuizOffered)
}

func (r *Registry) collectStaleQuizzes(rules Rules, now time.Time) {
	for token, quiz := range r.quizOffers {
		gone := now.After(quiz.expiresAt)
		if quiz.opened() {
			gone = now.After(quiz.deadline)
		}
		if !gone {
			continue
		}

		delete(r.quizOffers, token)

		entry, ok := r.callers[quiz.entrant]
		if !ok || entry.outstandingQuiz != token {
			continue
		}

		entry.outstandingQuiz = ""
		entry.nextQuizAt = now.Add(quizWait(rules))
		if entry.quizMisses == 0 {
			entry.nextQuizAt = now.Add(rules.Quiz.MissRetry)
		}
		entry.quizMisses++

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

func quizWait(rules Rules) time.Duration {
	return drawWindow(rules.Quiz.MinInterval, rules.Quiz.MaxInterval)
}
