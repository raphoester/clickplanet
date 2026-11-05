package get_budget_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type fullBuckets struct{}

func (fullBuckets) Peek(cpratelimit.Key) cpratelimit.State {
	return cpratelimit.State{Tokens: 10, Capacity: 10}
}

type flatPrice struct{}

func (flatPrice) Price(string) clicks.Price { return clicks.Price{Slowdown: 1} }

func TestTheBudgetSaysWhenTheRulesFreezeTheMap(t *testing.T) {
	buckets := clicks.NewBuckets(clicks.TestThrottle)
	for _, frozen := range []clicks.Freeze{true, false} {
		useCase := get_budget_usecase.New(fullBuckets{}, flatPrice{}, buckets, func() clicks.Freeze { return frozen })

		budget, ok := useCase.Execute(t.Context(), "fr")

		require.True(t, ok)
		assert.Equal(t, frozen, budget.Frozen)
	}
}
