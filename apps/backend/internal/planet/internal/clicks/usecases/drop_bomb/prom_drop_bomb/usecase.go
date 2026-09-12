// Package prom_drop_bomb counts where bombs land and how much they clear, to see whether the rings are set right.
package prom_drop_bomb

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/drop_bomb"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb.In) (clicks.Blast, error)
}

func New(implementation UseCase, registerer prometheus.Registerer) (*Decorator, error) {
	drops := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_bombs_dropped_total",
		Help: "Bomb drops, by outcome: land, sea or refused",
	}, []string{"outcome"})

	cleared := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bonus_bomb_tiles_cleared_total",
		Help: "Tiles bombs took away from whoever held them",
	})

	for _, collector := range []prometheus.Collector{drops, cleared} {
		if err := registerer.Register(collector); err != nil {
			return nil, fmt.Errorf("failed to register bomb collector: %w", err)
		}
	}

	return &Decorator{implementation: implementation, drops: drops, cleared: cleared}, nil
}

type Decorator struct {
	implementation UseCase
	drops          *prometheus.CounterVec
	cleared        prometheus.Counter
}

func (d *Decorator) Execute(ctx context.Context, in drop_bomb.In) (clicks.Blast, error) {
	blast, err := d.implementation.Execute(ctx, in)

	switch {
	case err != nil:
		d.drops.WithLabelValues("refused").Inc()
	case blast.Tile == 0:
		d.drops.WithLabelValues("sea").Inc()
	default:
		d.drops.WithLabelValues("land").Inc()
		d.cleared.Add(float64(len(blast.Cleared)))
	}

	return blast, err
}
