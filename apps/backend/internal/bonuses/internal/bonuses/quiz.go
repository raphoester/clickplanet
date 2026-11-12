package bonuses

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
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
	Outcome ratings.Outcome

	CorrectChoice int

	Reward Reward

	Subject string

	// What the ratings count of it besides its outcome: the question, and how long after the opening it landed.
	Question ratings.QuestionID
	After    time.Duration

	// Late is an answer to a question opened and lapsed a moment before, which the page's timeout sent over a slow
	// network or a clock behind the server's. To the player it is a quiz gone, as before; the ratings count it wrong.
	Late bool
}

// An answer lands at most this long past its question's deadline to be counted, late: the page's timeout reaches the
// server a trip after the deadline, and after the sweep that collected it on a slow network or a clock behind ours.
const lateAnswers = 10 * time.Second

type Questions interface {
	Draw(lean float64, fit quizzes.Fit) quizzes.Round
}

type pendingQuiz struct {
	entrant   Entrant
	scope     string
	round     quizzes.Round
	kind      Kind
	offeredAt time.Time

	expiresAt time.Time

	openedAt time.Time
	deadline time.Time
}

func (p *pendingQuiz) opened() bool { return !p.deadline.IsZero() }

func (r *Registry) OpenQuiz(token string, entrant Entrant) (Asked, bool) {
	now := r.clock.Now()
	window := r.rules().Quiz.AnswerWindow

	r.mu.Lock()
	defer r.mu.Unlock()

	quiz, ok := r.quizOffers[token]
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

	quiz, ok := r.quizOffers[token]
	if !ok {
		return r.answerLate(token, entrant, choice, now)
	}
	if quiz.entrant != entrant {
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
		Outcome:       quiz.outcome(choice, now),
		Question:      quiz.round.Question.ID,
		After:         now.Sub(quiz.openedAt),
	}

	if r.report.QuizAnswered != nil {
		r.report.QuizAnswered(quiz.scope, answered.Outcome == ratings.Right, now.Sub(quiz.offeredAt))
	}

	if answered.Outcome != ratings.Right {
		return answered, true
	}

	if known {
		entry.quizGrants = append(entry.quizGrants, now)
	}
	answered.Reward = Reward{Kind: quiz.kind, Amount: amountOf(quiz.kind, rules)}

	return answered, true
}

// answerLate counts an answer to a question opened and lapsed less than lateAnswers ago, once: the quiz is gone, and the
// answer is wrong and only counted.
func (r *Registry) answerLate(token string, entrant Entrant, choice int, now time.Time) (Answered, bool) {
	quiz, ok := r.lapsedQuizzes[token]
	if !ok || quiz.entrant != entrant || now.After(quiz.deadline.Add(lateAnswers)) {
		return Answered{}, false
	}

	delete(r.lapsedQuizzes, token)

	return Answered{
		CorrectChoice: quiz.round.Correct,
		Subject:       quiz.round.Question.Subject,
		Outcome:       quiz.outcome(choice, now),
		Question:      quiz.round.Question.ID,
		After:         now.Sub(quiz.openedAt),
		Late:          true,
	}, true
}

// The page's own timeout is a choice past the three it was shown.
func (p *pendingQuiz) outcome(choice int, now time.Time) ratings.Outcome {
	if choice < 0 || choice >= len(p.round.Options) {
		return ratings.TimedOut
	}
	if now.After(p.deadline) || !p.round.Correctly(choice) {
		return ratings.Wrong
	}
	return ratings.Right
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

	return !near(entry.outstanding, entry.lastOfferAt, entry.owedQuiz, rules, now)
}

func (r *Registry) offerQuiz(entrant Entrant, entry *caller, rules Rules, now time.Time, kind Kind) {
	token, err := newToken()
	if err != nil {
		return
	}

	round := r.bank.Draw(rules.Quiz.LeaderBias, rules.Quiz.Fit)
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
	entry.lastQuizAt = now
	entry.nextQuizAt = offer.ExpiresAt.Add(rules.Quiz.AnswerWindow).Add(quizWait(rules))

	entry.send(Event{Quiz: &offer})
	r.counted(r.report.QuizOffered)
}

func (r *Registry) collectStaleQuizzes(rules Rules, now time.Time) {
	for token, quiz := range r.lapsedQuizzes {
		if now.After(quiz.deadline.Add(lateAnswers)) {
			delete(r.lapsedQuizzes, token)
		}
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
		if quiz.opened() {
			r.lapsedQuizzes[token] = quiz
		}

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
