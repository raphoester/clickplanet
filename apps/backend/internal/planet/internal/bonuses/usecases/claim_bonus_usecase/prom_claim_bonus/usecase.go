package prom_claim_bonus

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus_usecase.In) (claim_bonus_usecase.Out, error)
}

type Counters struct {
	// Offered counts by kind and by the share the kind's band starts at.
	Offered func(kind bonuses.Kind, band float64)
	Lapsed  prometheus.Counter

	Caught prometheus.Histogram

	Foreign prometheus.Counter
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, Counters) {
	factory := promauto.With(registerer)

	claims := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_claims_total",
		Help: "Bonus boxes claimed, by outcome",
	}, []string{"outcome"})

	offers := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_offers_total",
		Help: "Bonus boxes put in front of a caller, by kind and by the share of the map their band starts at",
	}, []string{"kind", "band"})

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
		Counters{Offered: func(kind bonuses.Kind, band float64) {
			offers.WithLabelValues(string(kind), strconv.FormatFloat(band, 'g', -1, 64)).Inc()
		}, Lapsed: lapsed, Caught: caught, Foreign: foreign}
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
