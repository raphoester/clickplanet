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

	defended := factory.NewCounterVec(prometheus.CounterOpts{
		Name: "clicks_defended_total",
		Help: "Clicks that struck a defender rather than taking the tile, by the flag clicked",
	}, []string{
		"country_id",
	})

	return &UseCase{
		counter:        counter,
		defended:       defended,
		implementation: implementation,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	counter        *prometheus.CounterVec
	defended       *prometheus.CounterVec
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	status := "ok"
	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		status = "error"
		err = fmt.Errorf("failed to handle click: %w", err)
	}

	u.counter.WithLabelValues(in.CountryID, status).Inc()

	if err == nil && out.Outcome == clicks.Defended {
		u.defended.WithLabelValues(in.CountryID).Inc()
	}

	return out, err
}
