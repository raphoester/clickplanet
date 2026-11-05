package prom_enclose

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
)

func New(implementation enclose_click.Encloser, registerer prometheus.Registerer) *Encloser {
	factory := promauto.With(registerer)

	shapes := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_enclosures_total",
		Help: "Shapes closed with the enclose bonus",
	})

	tiles := factory.NewCounter(prometheus.CounterOpts{
		Name: "bonus_enclosed_tiles_total",
		Help: "Tiles taken inside shapes closed with the enclose bonus",
	})

	return &Encloser{implementation: implementation, shapes: shapes, tiles: tiles}
}

type Encloser struct {
	implementation enclose_click.Encloser
	shapes         prometheus.Counter
	tiles          prometheus.Counter
}

var _ enclose_click.Encloser = (*Encloser)(nil)

func (e *Encloser) Enclose(ctx context.Context, enclosure clicks.Enclosure, inside []uint32) error {
	if err := e.implementation.Enclose(ctx, enclosure, inside); err != nil {
		return err //nolint:wrapcheck // a pure delegation: the encloser already named what failed.
	}

	e.shapes.Inc()
	e.tiles.Add(float64(len(inside)))

	return nil
}
