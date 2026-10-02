package throttle_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type Limiter interface {
	TakeAll(n float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State)
}

type Pricer interface {
	Price(country string) clicks.Price
}

func New(implementation click_usecase.IUseCase, limiter Limiter, pricer Pricer, buckets clicks.Buckets) *UseCase {
	return &UseCase{implementation: implementation, limiter: limiter, pricer: pricer, buckets: buckets}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	limiter        Limiter
	pricer         Pricer
	buckets        clicks.Buckets
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	price := u.pricer.Price(in.CountryID)

	payer := clicks.PayerOf(ctx)
	allowed, states := u.limiter.TakeAll(1, u.buckets.Keys(payer, price)...)
	budget := u.buckets.BudgetOf(payer, states, price)

	if !allowed {
		return click_usecase.Out{Budget: budget, Limited: true}, clicks.ErrThrottled
	}

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = budget, true

	return out, err
}
