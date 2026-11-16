package prom_pay_caught_box

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, box bonuses.CaughtBox) (pay_caught_box_usecase.Out, error)
}

func New(implementation UseCase, registerer prometheus.Registerer) *Decorator {
	return &Decorator{
		implementation: implementation,
		paid: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
			Name: "bonus_box_rewards_total",
			Help: "Boxes caught the bonuses module drew a reward for, by outcome: paid, nothing when the wallet kept none of the kind drawn or no kind was weighed, or failed",
		}, []string{"outcome"}),
	}
}

type Decorator struct {
	implementation UseCase
	paid           *prometheus.CounterVec
}

func (d *Decorator) Execute(ctx context.Context, box bonuses.CaughtBox) (pay_caught_box_usecase.Out, error) {
	out, err := d.implementation.Execute(ctx, box)

	outcome := "paid"
	switch {
	case err != nil:
		outcome = "failed"
	case !out.Paid:
		outcome = "nothing"
	}
	d.paid.WithLabelValues(outcome).Inc()

	return out, err
}
