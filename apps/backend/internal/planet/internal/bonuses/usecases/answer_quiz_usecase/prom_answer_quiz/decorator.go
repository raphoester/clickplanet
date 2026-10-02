// Package prom_answer_quiz counts what the quizzes are doing.
//
// Three numbers are worth watching here and none of them is "how many were asked". **How many were
// opened against how many were offered** says whether the banner is noticed at all. **How many were
// answered right** is the bank's difficulty, and a rate near 1/3 means the questions are being
// guessed rather than known. **How long after the banner an answer landed** is the one that would
// show a script: a band of answers at the same fraction of a second, all correct, is not a person
// reading three choices.
package prom_answer_quiz

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in answer_quiz_usecase.In) (answer_quiz_usecase.Out, error)
}

// Counters are handed to the registry, which is what counts an offer, a banner nobody opened, and
// how long an answer took. The address is never a label, for the reason the boxes' are not:
// unbounded cardinality, and personal data in every scrape.
type Counters struct {
	Offered prometheus.Counter
	Lapsed  prometheus.Counter

	// Answered is how long after the banner an answer landed, by whether it was right.
	Answered *prometheus.HistogramVec
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, Counters) {
	factory := promauto.With(registerer)

	opened := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "quiz_answers_total",
		Help: "Quizzes answered, by outcome",
	}, []string{"outcome"})

	offers := factory.NewCounter(prometheus.CounterOpts{
		Name: "quiz_offers_total",
		Help: "Quizzes put in front of a caller",
	})

	lapsed := factory.NewCounter(prometheus.CounterOpts{
		Name: "quiz_lapsed_total",
		Help: "Quizzes that were offered and never answered",
	})

	// The banner can sit for its whole offerTTL before it is opened, so the buckets run well past
	// the answer window: what is interesting is the shape near the bottom.
	answered := factory.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "quiz_answer_seconds",
		Help:    "How long after a quiz was offered it was answered",
		Buckets: []float64{0.5, 1, 1.5, 2, 3, 4, 5, 7, 10, 15, 20, 30},
	}, []string{"correct"})

	return &Decorator{implementation: implementation, answers: opened},
		Counters{Offered: offers, Lapsed: lapsed, Answered: answered}
}

type Decorator struct {
	implementation UseCase
	answers        *prometheus.CounterVec
}

func (d *Decorator) Execute(
	ctx context.Context,
	in answer_quiz_usecase.In,
) (answer_quiz_usecase.Out, error) {
	out, err := d.implementation.Execute(ctx, in)

	// "refused" is a quiz that was not this caller's to answer. Wrong is not refused: it landed,
	// and it is the number the bank is judged on.
	outcome := "wrong"
	switch {
	case err != nil:
		outcome = "refused"
	case out.Correct:
		outcome = "correct"
	}
	d.answers.WithLabelValues(outcome).Inc()

	return out, err
}
