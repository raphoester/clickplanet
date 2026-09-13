package get_budget_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	state   toll.Budget
	limited bool
	country *string
}

func (s stubUseCase) Execute(_ context.Context, country string) (toll.Budget, bool) {
	if s.country != nil {
		*s.country = country
	}

	return s.state, s.limited
}

func getBudget(t *testing.T, useCase stubUseCase) *planetv1.ClickBudget {
	t.Helper()

	res, err := get_budget_handler.New(useCase).
		GetBudget(t.Context(), connect.NewRequest(&planetv1.GetBudgetRequest{CountryId: "bg"}))
	require.NoError(t, err)

	return res.Msg.GetBudget()
}

func TestGetBudgetMapsTheReading(t *testing.T) {
	budget := getBudget(t, stubUseCase{
		state: toll.Budget{
			State: cpratelimit.State{Tokens: 7.25, Capacity: 10, PerSecond: 1.5},
			Price: toll.Price{Cost: 2, Share: 0.3, NextShare: 0.5, NextCost: 4},
		},
		limited: true,
	})

	require.NotNil(t, budget)
	assert.InDelta(t, 7.25, budget.GetTokens(), 1e-9)
	assert.Equal(t, uint32(10), budget.GetCapacity())
	assert.InDelta(t, 1.5, budget.GetRefillPerSecond(), 1e-9)
	assert.Equal(t, uint32(2), budget.GetCost())
	assert.InDelta(t, 0.3, budget.GetShare(), 1e-9)
	assert.InDelta(t, 0.5, budget.GetNextShare(), 1e-9)
	assert.Equal(t, uint32(4), budget.GetNextCost())
}

func TestGetBudgetPricesTheCountryAskedAbout(t *testing.T) {
	var country string
	getBudget(t, stubUseCase{limited: true, country: &country})

	assert.Equal(t, "bg", country)
}

// An unthrottled server promising an allowance of zero would have every client
// show an empty meter and refuse to click.
func TestGetBudgetIsAbsentWhenNothingThrottles(t *testing.T) {
	require.Nil(t, getBudget(t, stubUseCase{limited: false}))
}
