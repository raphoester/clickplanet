package planetv1controller

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
)

type fakeBudget struct {
	remaining int
	charged   []string
}

func (b *fakeBudget) Spend(id string) bool {
	b.charged = append(b.charged, id)

	if b.remaining <= 0 {
		return false
	}
	b.remaining--
	return true
}

type budgetResult struct {
	ran      bool
	registry prometheus.Gatherer
	err      error
}

func spendBudget(t *testing.T, budget ClickBudget, procedure string, sessionID string) budgetResult {
	t.Helper()

	registry := prometheus.NewRegistry()
	interceptor, err := NewClickBudgetInterceptor(budget, registry)
	require.NoError(t, err)

	result := budgetResult{registry: registry}
	next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		result.ran = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	ctx := context.Background()
	if sessionID != "" {
		ctx = ctxutil.AddSessionIDToContext(ctx, sessionID)
	}

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}}

	_, result.err = interceptor.WrapUnary(next)(ctx, req)

	return result
}

func TestAClickIsChargedToTheSessionThatCarriedIt(t *testing.T) {
	budget := &fakeBudget{remaining: 1}

	result := spendBudget(t, budget, planetv1connect.ClickServiceClickProcedure, "abcd1234")

	require.NoError(t, result.err)
	require.True(t, result.ran)
	require.Equal(t, []string{"abcd1234"}, budget.charged)
	require.Equal(t, 0.0, budgetExhausted(t, result.registry))
}

func TestAClickIsRefusedOnceItsSessionHasSpentItsBudget(t *testing.T) {
	budget := &fakeBudget{remaining: 0}

	result := spendBudget(t, budget, planetv1connect.ClickServiceClickProcedure, "abcd1234")

	// Unauthenticated rather than a code of its own: it asks the client for the
	// same thing a lapsed token does, and the client already answers it.
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(result.err))
	require.ErrorIs(t, result.err, ErrBudgetSpent)
	require.False(t, result.ran, "a refused click must not reach the domain")
	require.Equal(t, 1.0, budgetExhausted(t, result.registry))
}

func TestACallerWithNoSessionIsNotCharged(t *testing.T) {
	// The state of every caller when sessions are off or not yet enforced.
	// Refusing here would take the click path down with the budget.
	budget := &fakeBudget{remaining: 0}

	result := spendBudget(t, budget, planetv1connect.ClickServiceClickProcedure, "")

	require.NoError(t, result.err)
	require.True(t, result.ran)
	require.Empty(t, budget.charged, "the budget should not be consulted without a session")
}

func TestOtherProceduresAreNotCharged(t *testing.T) {
	budget := &fakeBudget{remaining: 0}

	result := spendBudget(t, budget, planetv1connect.ClickServiceGetMapProcedure, "abcd1234")

	require.NoError(t, result.err)
	require.True(t, result.ran, "reads are not clicks and cost no budget")
	require.Empty(t, budget.charged)
}

func budgetExhausted(t *testing.T, gatherer prometheus.Gatherer) float64 {
	t.Helper()

	families, err := gatherer.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != "click_budget_exhausted" {
			continue
		}
		return family.GetMetric()[0].GetCounter().GetValue()
	}

	return 0
}
