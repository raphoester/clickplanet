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

// Discount is what a click on the tile takes from the bank: a whole one, or less inside a rally's ring of its flag. A
// failed read answers a whole one with its error.
type Discount interface {
	Cost(ctx context.Context, flag string, tile uint32) (float64, error)
}

func New(
	implementation click_usecase.IUseCase, limiter Limiter, pricer Pricer, discount Discount, buckets clicks.Buckets,
) *UseCase {
	return &UseCase{implementation: implementation, limiter: limiter, pricer: pricer, discount: discount, buckets: buckets}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	limiter        Limiter
	pricer         Pricer
	discount       Discount
	buckets        clicks.Buckets
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	price := u.pricer.Price(in.CountryID)
	// A failed read was told by the discount's log, and prices the click whole.
	cost, _ := u.discount.Cost(ctx, in.CountryID, in.TileID)

	payer := clicks.PayerOf(ctx)
	allowed, states := u.limiter.TakeAll(cost, u.buckets.Keys(payer, price)...)
	budget := u.buckets.BudgetOf(payer, states, price)
	budget.Spent = cost

	if !allowed {
		return click_usecase.Out{Budget: budget, Limited: true}, clicks.ErrThrottled
	}

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = budget, true

	return out, err
}
