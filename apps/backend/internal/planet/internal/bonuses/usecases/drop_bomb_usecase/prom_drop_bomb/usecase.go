// Package prom_drop_bomb counts where bombs land and how much they clear, to see whether the rings are set right.
package prom_drop_bomb

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
}

func New(implementation UseCase, registerer prometheus.Registerer) *Decorator {
	factory := promauto.With(registerer)

	drops := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_bombs_dropped_total",
		Help: "Bomb drops, by outcome: land, sea, refused or shadowbanned",
	}, []string{"outcome"})

	cleared := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_bomb_tiles_cleared_total",
		Help: "Tiles bombs took away from whoever held them",
	})

	return &Decorator{implementation: implementation, drops: drops, cleared: cleared}
}

type Decorator struct {
	implementation UseCase
	drops          *prometheus.CounterVec
	cleared        prometheus.Counter
}

func (d *Decorator) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	blast, err := d.implementation.Execute(ctx, in)

	switch {
	case err != nil:
		d.drops.WithLabelValues("refused").Inc()
	case in.Dud:
		d.drops.WithLabelValues("shadowbanned").Inc()
	case blast.Tile == 0:
		d.drops.WithLabelValues("sea").Inc()
	default:
		d.drops.WithLabelValues("land").Inc()
		d.cleared.Add(float64(len(blast.Cleared)))
	}

	return blast, err
}
