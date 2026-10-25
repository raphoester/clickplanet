package prom_throttle

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

func New(implementation click_usecase.IUseCase, registerer prometheus.Registerer) *UseCase {
	refused := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "throttled_clicks",
		Help: "Clicks refused because a bucket they spend from was empty",
	})

	return &UseCase{implementation: implementation, refused: refused}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	refused        prometheus.Counter
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if errors.Is(err, clicks.ErrThrottled) {
		u.refused.Inc()
	}

	return out, err //nolint:wrapcheck // a decorator that counts adds no sentence.
}
