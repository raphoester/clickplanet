// Package prom_enclose counts the shapes the enclose bonus closes and the tiles
// they take, to see whether the shape and size limits are set right.
//
// It wraps the publisher rather than the click: every shape closed is published
// exactly once, and a click that closed nothing is not an enclosure to count.
package prom_enclose

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
)

func New(implementation enclose_click.Publisher, registerer prometheus.Registerer) *Publisher {
	factory := promauto.With(registerer)

	shapes := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_enclosures_total",
		Help: "Shapes closed with the enclose bonus",
	})

	tiles := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_enclosed_tiles_total",
		Help: "Tiles taken inside shapes closed with the enclose bonus",
	})

	return &Publisher{implementation: implementation, shapes: shapes, tiles: tiles}
}

type Publisher struct {
	implementation enclose_click.Publisher
	shapes         prometheus.Counter
	tiles          prometheus.Counter
}

var _ enclose_click.Publisher = (*Publisher)(nil)

func (p *Publisher) PublishEnclosed(scope string, enclosed bonus.Enclosed) {
	p.shapes.Inc()
	p.tiles.Add(float64(len(enclosed.Filled)))

	p.implementation.PublishEnclosed(scope, enclosed)
}
