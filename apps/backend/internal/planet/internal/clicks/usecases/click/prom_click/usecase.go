// Package prom_click counts what the click use case decided. It is a decorator
// and not a line inside the use case, so the measuring can be left out of a
// process that does not want it without the rule changing.
package prom_click

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func New(
	implementation click.IUseCase,
	registerer prometheus.Registerer,
) (*UseCase, error) {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "clicks",
		Help: "Registered clicks",
	}, []string{
		"source_ip",
		"country_id",
		"status",
	})

	if err := registerer.Register(histogram); err != nil {
		return nil, fmt.Errorf("failed to register histogram: %w", err)
	}

	return &UseCase{
		histogram:      histogram,
		implementation: implementation,
	}, nil
}

type UseCase struct {
	implementation click.IUseCase
	histogram      *prometheus.HistogramVec
}

func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	sourceIP := cpctx.GetSourceIP(ctx)
	status := "ok"
	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		status = "error"
		err = fmt.Errorf("failed to handle click: %w", err)
	}

	u.histogram.WithLabelValues(
		sourceIP,
		in.CountryID,
		status,
	).Observe(1)

	return out, err
}
