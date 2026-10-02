package get_budget_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type ClickBudgetReader interface {
	Peek(key cpratelimit.Key) cpratelimit.State
}

type Pricer interface {
	PriceFor(ctx context.Context, payer clicks.Payer, country string) (clicks.Price, error)
}

func New(budgets ClickBudgetReader, pricer Pricer, buckets clicks.Buckets) *UseCase {
	return &UseCase{budgets: budgets, pricer: pricer, buckets: buckets}
}

type UseCase struct {
	budgets ClickBudgetReader
	pricer  Pricer
	buckets clicks.Buckets
}

func (u *UseCase) Execute(ctx context.Context, country string) (clicks.Budget, bool, error) {
	if u.budgets == nil {
		return clicks.Budget{}, false, nil
	}

	payer := clicks.PayerOf(ctx)
	price, err := u.pricer.PriceFor(ctx, payer, country)
	if err != nil {
		return clicks.Budget{}, false, fmt.Errorf("failed to price the budget: %w", err)
	}

	keys := u.buckets.Keys(payer, price)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.budgets.Peek(key)
	}

	return u.buckets.BudgetOf(payer, states, price), true, nil
}
