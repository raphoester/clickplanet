package get_budget_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	state   cpratelimit.State
	limited bool
}

func (s stubUseCase) Execute(context.Context) (cpratelimit.State, bool) {
	return s.state, s.limited
}

func getBudget(t *testing.T, useCase stubUseCase) *planetv1.ClickBudget {
	t.Helper()

	res, err := get_budget_handler.New(useCase).
		GetBudget(t.Context(), connect.NewRequest(&planetv1.GetBudgetRequest{}))
	require.NoError(t, err)

	return res.Msg.GetBudget()
}

func TestGetBudgetMapsTheReading(t *testing.T) {
	budget := getBudget(t, stubUseCase{
		state:   cpratelimit.State{Tokens: 7.25, Capacity: 10, PerSecond: 1.5},
		limited: true,
	})

	require.NotNil(t, budget)
	assert.InDelta(t, 7.25, budget.GetTokens(), 1e-9)
	assert.Equal(t, uint32(10), budget.GetCapacity())
	assert.InDelta(t, 1.5, budget.GetRefillPerSecond(), 1e-9)
}

// An unthrottled server promising an allowance of zero would have every client
// show an empty meter and refuse to click.
func TestGetBudgetIsAbsentWhenNothingThrottles(t *testing.T) {
	require.Nil(t, getBudget(t, stubUseCase{limited: false}))
}
