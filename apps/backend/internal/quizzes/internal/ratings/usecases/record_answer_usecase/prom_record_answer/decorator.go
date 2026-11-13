package prom_record_answer

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

type UseCase interface {
	Execute(ctx context.Context, answer ratings.Answer) error
}

func New(implementation UseCase, registerer prometheus.Registerer) *Decorator {
	return &Decorator{implementation: implementation, recorded: promauto.With(registerer).NewCounterVec(
		prometheus.CounterOpts{
			Name: "quiz_answers_recorded_total",
			Help: "Quiz answers counted for their question's rating, by outcome: recorded, or failed",
		}, []string{"outcome"})}
}

type Decorator struct {
	implementation UseCase
	recorded       *prometheus.CounterVec
}

func (d *Decorator) Execute(ctx context.Context, answer ratings.Answer) error {
	err := d.implementation.Execute(ctx, answer)

	outcome := "recorded"
	if err != nil {
		outcome = "failed"
	}
	d.recorded.WithLabelValues(outcome).Inc()

	return err //nolint:wrapcheck // a decorator counts, it adds no sentence.
}
