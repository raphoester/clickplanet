package planetv1controller

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
)

type switchedRules struct{ frozen bool }

func (r *switchedRules) Frozen() bool { return r.frozen }

func call(t *testing.T, interceptor connect.Interceptor, procedure string) (bool, error) {
	t.Helper()

	reached := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		reached = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{spec: connect.Spec{Procedure: procedure}})

	return reached, err
}

func frozenCall(t *testing.T, rules MapRules, procedure string) (bool, error) {
	t.Helper()

	return call(t, NewFreezeInterceptor(rules, prometheus.NewRegistry()), procedure)
}

func TestAFrozenMapRefusesEveryWriteAsFrozen(t *testing.T) {
	for _, procedure := range []string{
		planetv1connect.ClickServiceClickProcedure,
		planetv1connect.ClickServiceDropBombProcedure,
		planetv1connect.ClickServiceUseRefillProcedure,
		planetv1connect.ClickServicePlaceShieldProcedure,
		planetv1connect.ClickServiceClaimBonusProcedure,
		planetv1connect.ClickServiceAnswerQuizProcedure,
	} {
		t.Run(procedure, func(t *testing.T) {
			reached, err := frozenCall(t, &switchedRules{frozen: true}, procedure)

			require.False(t, reached, "nothing behind the interceptor runs")
			var connectErr *connect.Error
			require.ErrorAs(t, err, &connectErr)
			assert.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
			require.Len(t, connectErr.Details(), 1)
			detail, err := connectErr.Details()[0].Value()
			require.NoError(t, err)
			assert.IsType(t, &planetv1.MapFrozen{}, detail, "the detail is what tells it from a full bank")
		})
	}
}

func TestAFrozenMapIsStillReadAndPledgedTo(t *testing.T) {
	for _, procedure := range []string{
		planetv1connect.ClickServiceGetBudgetProcedure,
		planetv1connect.ClickServiceGetMapProcedure,
		planetv1connect.ClickServiceGetChargesProcedure,
		planetv1connect.ClickServiceOpenQuizProcedure,
		planetv1connect.ClickServicePledgeAllegianceProcedure,
	} {
		reached, err := frozenCall(t, &switchedRules{frozen: true}, procedure)

		require.NoError(t, err, procedure)
		assert.True(t, reached, procedure)
	}
}

func TestAnOpenMapLetsEveryWriteThrough(t *testing.T) {
	reached, err := frozenCall(t, &switchedRules{}, planetv1connect.ClickServiceClickProcedure)

	require.NoError(t, err)
	assert.True(t, reached)
}

func TestOnlyTheClicksAFrozenMapRefusesAreCountedAsFrozenClicks(t *testing.T) {
	registry := prometheus.NewRegistry()
	rules := &switchedRules{frozen: true}
	interceptor := NewFreezeInterceptor(rules, registry)

	_, _ = call(t, interceptor, planetv1connect.ClickServiceClickProcedure)
	_, _ = call(t, interceptor, planetv1connect.ClickServiceDropBombProcedure)
	rules.frozen = false
	_, _ = call(t, interceptor, planetv1connect.ClickServiceClickProcedure)

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP frozen_clicks Clicks refused because the rules in force freeze the map
# TYPE frozen_clicks counter
frozen_clicks 1
`), "frozen_clicks"))
}

func TestAFrozenClickSpendsNoTokenAndAThawedOneDoes(t *testing.T) {
	rules := &switchedRules{frozen: true}
	limiter := &fakeLimiter{allow: true}
	server := clickServerWith(t, throttle_click.New(stubService{}, limiter, onePrice, buckets), nil, connect.WithInterceptors(
		errorNet(),
		NewFreezeInterceptor(rules, prometheus.NewRegistry()),
	))

	require.Equal(t, http.StatusBadRequest, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, limiter.keys, "the frozen click never reached the bucket")

	rules.frozen = false
	require.Equal(t, http.StatusOK, clickStatus(t, server, "1.2.3.4"))
	assert.NotEmpty(t, limiter.keys)
}
