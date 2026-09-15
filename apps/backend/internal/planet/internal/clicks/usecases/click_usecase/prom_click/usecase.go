// Package prom_click counts what the click use case decided. It is a decorator
// and not a line inside the use case, so the measuring can be left out of a
// process that does not want it without the rule changing.
package prom_click

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

func New(
	implementation click_usecase.IUseCase,
	registerer prometheus.Registerer,
) *UseCase {
	counter := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_total",
		Help: "Clicks that reached the rule, by country and outcome",
	}, []string{
		"country_id",
		"status",
	})

	return &UseCase{
		counter:        counter,
		implementation: implementation,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	counter        *prometheus.CounterVec
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	status := "ok"
	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		status = "error"
		err = fmt.Errorf("failed to handle click: %w", err)
	}

	u.counter.WithLabelValues(in.CountryID, status).Inc()

	return out, err
}
