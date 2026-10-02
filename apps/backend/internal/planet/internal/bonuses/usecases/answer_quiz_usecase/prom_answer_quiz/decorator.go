package prom_answer_quiz

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in answer_quiz_usecase.In) (answer_quiz_usecase.Out, error)
}

type Counters struct {
	// Offered counts by kind and by the share the kind's band starts at.
	Offered func(kind bonuses.Kind, band float64)
	Lapsed  prometheus.Counter

	Answered *prometheus.HistogramVec
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, Counters) {
	factory := promauto.With(registerer)

	opened := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "quiz_answers_total",
		Help: "Quizzes answered, by outcome",
	}, []string{"outcome"})

	offers := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "quiz_offers_total",
		Help: "Quizzes put in front of a caller, by the kind a right answer wins and the share its band starts at",
	}, []string{"kind", "band"})

	lapsed := factory.NewCounter(prometheus.CounterOpts{
		Name: "quiz_lapsed_total",
		Help: "Quizzes that were offered and never answered",
	})

	answered := factory.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "quiz_answer_seconds",
		Help:    "How long after a quiz was offered it was answered",
		Buckets: []float64{0.5, 1, 1.5, 2, 3, 4, 5, 7, 10, 15, 20, 30},
	}, []string{"correct"})

	return &Decorator{implementation: implementation, answers: opened},
		Counters{Offered: func(kind bonuses.Kind, band float64) {
			offers.WithLabelValues(string(kind), strconv.FormatFloat(band, 'g', -1, 64)).Inc()
		}, Lapsed: lapsed, Answered: answered}
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
