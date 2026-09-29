// Package prom_claim_bonus counts what the boxes are doing, for a box with no
// dashboard: offered against caught is the only way to see whether the pacing
// and the flight time are set anywhere near right.
package prom_claim_bonus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus_usecase.In) (claim_bonus_usecase.Out, error)
}

// Counters are handed to the registry as well, which is what counts an offer
// and a box nobody took. The address is never a label: unbounded cardinality,
// and personal data in every scrape.
type Counters struct {
	Offered prometheus.Counter
	Lapsed  prometheus.Counter

	// Caught is how long after the offer each box was claimed. The raw bucket
	// counts are what show a band of callers claiming before a person could
	// have found the box.
	Caught prometheus.Histogram

	// Foreign counts refused claims of a box offered to another caller or to nobody.
	Foreign prometheus.Counter
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, Counters) {
	factory := promauto.With(registerer)

	claims := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_claims_total",
		Help: "Bonus boxes claimed, by outcome",
	}, []string{"outcome"})

	offers := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_offers_total",
		Help: "Bonus boxes put in front of a caller",
	})

	lapsed := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_lapsed_total",
		Help: "Bonus boxes that were offered and not caught",
	})

	caught := factory.NewHistogram(prometheus.HistogramOpts{
		Name:    "bonus_catch_seconds",
		Help:    "How long after a bonus box was offered it was claimed",
		Buckets: []float64{0.25, 0.5, 1, 1.5, 2, 3, 4, 6, 8, 10, 12, 15},
	})

	foreign := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_claims_foreign_total",
		Help: "Refused claims of a bonus box that was offered to another caller, or to nobody",
	})

	return &Decorator{implementation: implementation, claims: claims},
		Counters{Offered: offers, Lapsed: lapsed, Caught: caught, Foreign: foreign}
}

type Decorator struct {
	implementation UseCase
	claims         *prometheus.CounterVec
}

func (d *Decorator) Execute(ctx context.Context, in claim_bonus_usecase.In) (claim_bonus_usecase.Out, error) {
	out, err := d.implementation.Execute(ctx, in)

	outcome := "granted"
	if err != nil {
		outcome = "refused"
	}
	d.claims.WithLabelValues(outcome).Inc()

	return out, err
}
