// Package prom_click counts what the click use case decided. It is a decorator
// and not a line inside the use case, so the measuring can be left out of a
// process that does not want it without the rule changing.
package prom_click

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

func New(
	implementation click_usecase.IUseCase,
	registerer prometheus.Registerer,
) *UseCase {
	factory := promauto.With(registerer)

	counter := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_total",
		Help: "Clicks that reached the rule, by country and outcome",
	}, []string{
		"country_id",
		"status",
	})

	// Apart from clicks_total, so its series and every panel reading status="ok" stay as they were: a clear is
	// an accepted click, and counted there too.
	cleared := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_cleared_total",
		Help: "Clicks that cleared a tile on its own country's ground rather than taking it, by the flag clicked",
	}, []string{
		"country_id",
	})

	return &UseCase{
		counter:        counter,
		cleared:        cleared,
		implementation: implementation,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	counter        *prometheus.CounterVec
	cleared        *prometheus.CounterVec
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	status := "ok"
	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		status = "error"
		err = fmt.Errorf("failed to handle click: %w", err)
	}

	u.counter.WithLabelValues(in.CountryID, status).Inc()

	if err == nil && out.Outcome == clicks.Cleared {
		u.cleared.WithLabelValues(in.CountryID).Inc()
	}

	return out, err
}
