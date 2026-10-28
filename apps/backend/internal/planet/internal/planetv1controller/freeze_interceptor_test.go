package planetv1controller

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func frozenCall(t *testing.T, frozen clicks.Freeze, procedure string) (bool, error) {
	t.Helper()

	ran := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		ran = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})
	interceptor := NewFreezeInterceptor(func() clicks.Freeze { return frozen })

	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{spec: connect.Spec{Procedure: procedure}})
	return ran, err
}

func TestAFrozenMapRefusesEveryPlayBeforeItSpendsAnything(t *testing.T) {
	for _, procedure := range []string{
		planetv1connect.ClickServiceClickProcedure,
		planetv1connect.ClickServiceDropBombProcedure,
		planetv1connect.ClickServicePlaceShieldProcedure,
		planetv1connect.ClickServiceUseRefillProcedure,
		planetv1connect.ClickServiceClaimBonusProcedure,
		planetv1connect.ClickServiceOpenQuizProcedure,
		planetv1connect.ClickServiceAnswerQuizProcedure,
	} {
		t.Run(procedure, func(t *testing.T) {
			ran, err := frozenCall(t, true, procedure)

			require.False(t, ran)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.ErrorIs(t, err, clicks.ErrFrozen)
			var refused *connect.Error
			require.ErrorAs(t, err, &refused)
			require.Len(t, refused.Details(), 1)
			detail, err := refused.Details()[0].Value()
			require.NoError(t, err)
			assert.IsType(t, &planetv1.FrozenRefusal{}, detail)
		})
	}
}

func TestAFrozenMapStillLoadsAndStreams(t *testing.T) {
	for _, procedure := range []string{
		planetv1connect.ClickServiceGetMapProcedure,
		planetv1connect.ClickServiceGetBudgetProcedure,
		planetv1connect.ClickServiceGetHoldingsProcedure,
		planetv1connect.ClickServiceGetTileHolderProcedure,
		planetv1connect.ClickServicePledgeAllegianceProcedure,
	} {
		ran, err := frozenCall(t, true, procedure)
		require.NoError(t, err, procedure)
		assert.True(t, ran, procedure)
	}
}

func TestAMapNotFrozenTakesPlay(t *testing.T) {
	ran, err := frozenCall(t, false, planetv1connect.ClickServiceClickProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
}
