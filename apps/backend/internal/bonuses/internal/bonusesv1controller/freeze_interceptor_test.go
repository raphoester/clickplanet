package bonusesv1controller_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller"
)

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec { return r.spec }

func call(t *testing.T, frozen bonusesv1controller.Frozen, procedure string) (bool, error) {
	t.Helper()

	ran := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		ran = true
		return connect.NewResponse(&bonusesv1.ClaimBonusResponse{}), nil
	})

	_, err := bonusesv1controller.NewFreezeInterceptor(frozen).WrapUnary(next)(t.Context(),
		fakeRequest{spec: connect.Spec{Procedure: procedure}})
	return ran, err
}

func always(frozen bool) bonusesv1controller.Frozen {
	return func(context.Context) (bool, error) { return frozen, nil }
}

func TestWhileFrozenNoBoxOrQuizIsTakenBeforeItIsSpent(t *testing.T) {
	for _, procedure := range []string{
		bonusesv1connect.BonusServiceClaimBonusProcedure,
		bonusesv1connect.BonusServiceOpenQuizProcedure,
		bonusesv1connect.BonusServiceAnswerQuizProcedure,
	} {
		t.Run(procedure, func(t *testing.T) {
			ran, err := call(t, always(true), procedure)

			require.False(t, ran)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.ErrorIs(t, err, bonuses.ErrFrozen)
			var refused *connect.Error
			require.ErrorAs(t, err, &refused)
			require.Len(t, refused.Details(), 1)
			detail, err := refused.Details()[0].Value()
			require.NoError(t, err)
			assert.IsType(t, &bonusesv1.FrozenRefusal{}, detail)
		})
	}
}

func TestWhileFrozenTheReadsGoOn(t *testing.T) {
	asked := false
	frozen := func(context.Context) (bool, error) { asked = true; return true, nil }

	ran, err := call(t, frozen, bonusesv1connect.BonusServiceGetGrantRulesProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
	assert.False(t, asked, "a read asks planet nothing")
}

func TestNotFrozenABoxIsTaken(t *testing.T) {
	ran, err := call(t, always(false), bonusesv1connect.BonusServiceClaimBonusProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
}

func TestAFreezeNobodyCanTellTakesNothing(t *testing.T) {
	ran, err := call(t, func(context.Context) (bool, error) { return false, errors.New("planet is gone") },
		bonusesv1connect.BonusServiceClaimBonusProcedure)

	require.Error(t, err)
	assert.False(t, ran)
}
