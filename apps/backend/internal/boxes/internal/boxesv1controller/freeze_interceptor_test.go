package boxesv1controller_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller"
)

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec { return r.spec }

func call(t *testing.T, frozen boxesv1controller.Frozen, procedure string) (bool, error) {
	t.Helper()

	ran := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		ran = true
		return connect.NewResponse(&boxesv1.ClaimBoxResponse{}), nil
	})

	_, err := boxesv1controller.NewFreezeInterceptor(frozen).WrapUnary(next)(t.Context(),
		fakeRequest{spec: connect.Spec{Procedure: procedure}})
	return ran, err
}

func always(frozen bool) boxesv1controller.Frozen {
	return func(context.Context) (bool, error) { return frozen, nil }
}

func TestWhileFrozenNoBoxIsTakenBeforeItIsSpent(t *testing.T) {
	for _, procedure := range []string{
		boxesv1connect.BoxServiceClaimBoxProcedure,
	} {
		t.Run(procedure, func(t *testing.T) {
			ran, err := call(t, always(true), procedure)

			require.False(t, ran)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.ErrorIs(t, err, boxes.ErrFrozen)
			var refused *connect.Error
			require.ErrorAs(t, err, &refused)
			require.Len(t, refused.Details(), 1)
			detail, err := refused.Details()[0].Value()
			require.NoError(t, err)
			assert.IsType(t, &boxesv1.FrozenRefusal{}, detail)
		})
	}
}

func TestWhileFrozenTheStreamGoesOn(t *testing.T) {
	asked := false
	frozen := func(context.Context) (bool, error) { asked = true; return true, nil }

	ran, err := call(t, frozen, boxesv1connect.BoxServiceListenForEventsProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
	assert.False(t, asked, "a stream asks planet nothing")
}

func TestNotFrozenABoxIsTaken(t *testing.T) {
	ran, err := call(t, always(false), boxesv1connect.BoxServiceClaimBoxProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
}

func TestAFreezeNobodyCanTellTakesNothing(t *testing.T) {
	ran, err := call(t, func(context.Context) (bool, error) { return false, errors.New("planet is gone") },
		boxesv1connect.BoxServiceClaimBoxProcedure)

	require.Error(t, err)
	assert.False(t, ran)
}
