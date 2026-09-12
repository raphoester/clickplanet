// Package prom_claim_bonus counts what the boxes are doing, for a box with no
// dashboard: offered against caught is the only way to see whether the pacing
// and the flight time are set anywhere near right.
package prom_claim_bonus

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus.In) (claim_bonus.Out, error)
}

// Counters are handed to the registry as well, which is what counts an offer
// and a box nobody took. The address is never a label: unbounded cardinality,
// and personal data in every scrape.
type Counters struct {
	Offered prometheus.Counter
	Lapsed  prometheus.Counter
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, Counters, error) {
	claims := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_claims_total",
		Help: "Bonus boxes claimed, by outcome",
	}, []string{"outcome"})

	offers := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bonus_offers_total",
		Help: "Bonus boxes put in front of a caller",
	})

	lapsed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bonus_lapsed_total",
		Help: "Bonus boxes that were offered and not caught",
	})

	for _, collector := range []prometheus.Collector{claims, offers, lapsed} {
		if err := registerer.Register(collector); err != nil {
			return nil, Counters{}, fmt.Errorf("failed to register bonus collector: %w", err)
		}
	}

	return &Decorator{implementation: implementation, claims: claims},
		Counters{Offered: offers, Lapsed: lapsed},
		nil
}

type Decorator struct {
	implementation UseCase
	claims         *prometheus.CounterVec
}

func (d *Decorator) Execute(ctx context.Context, in claim_bonus.In) (claim_bonus.Out, error) {
	out, err := d.implementation.Execute(ctx, in)

	outcome := "granted"
	if err != nil {
		outcome = "refused"
	}
	d.claims.WithLabelValues(outcome).Inc()

	return out, err
}
