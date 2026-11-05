package refusal_click

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

func New(implementation click_usecase.IUseCase, refusals Refusals, registerer prometheus.Registerer) *UseCase {
	refused := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "refused_clicks",
		Help: "Clicks refused because their caller is refused",
	})

	return &UseCase{implementation: implementation, refusals: refusals, refused: refused}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	refusals       Refusals
	refused        prometheus.Counter
}

// A refusal that cannot be read refuses nothing: postgres down lets a refused caller click, not every caller fail.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if refusal, refused, err := u.refusals.Refusal(ctx, clicks.PayerOf(ctx)); err == nil && refused {
		u.refused.Inc()
		return click_usecase.Out{}, refusal
	}

	return u.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
