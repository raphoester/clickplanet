package quizzesv1controller_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller"
)

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec { return r.spec }

func call(t *testing.T, frozen quizzesv1controller.Frozen, procedure string) (bool, error) {
	t.Helper()

	ran := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		ran = true
		return connect.NewResponse(&quizzesv1.OpenQuizResponse{}), nil
	})

	_, err := quizzesv1controller.NewFreezeInterceptor(frozen).WrapUnary(next)(t.Context(),
		fakeRequest{spec: connect.Spec{Procedure: procedure}})
	return ran, err
}

func always(frozen bool) quizzesv1controller.Frozen {
	return func(context.Context) (bool, error) { return frozen, nil }
}

func TestWhileFrozenNoQuizIsTakenBeforeItIsSpent(t *testing.T) {
	for _, procedure := range []string{
		quizzesv1connect.QuizServiceOpenQuizProcedure,
		quizzesv1connect.QuizServiceAnswerQuizProcedure,
	} {
		t.Run(procedure, func(t *testing.T) {
			ran, err := call(t, always(true), procedure)

			require.False(t, ran)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
			require.ErrorIs(t, err, quizzes.ErrFrozen)
			var refused *connect.Error
			require.ErrorAs(t, err, &refused)
			require.Len(t, refused.Details(), 1)
			detail, err := refused.Details()[0].Value()
			require.NoError(t, err)
			assert.IsType(t, &quizzesv1.FrozenRefusal{}, detail)
		})
	}
}

func TestWhileFrozenTheStreamGoesOn(t *testing.T) {
	asked := false
	frozen := func(context.Context) (bool, error) { asked = true; return true, nil }

	ran, err := call(t, frozen, quizzesv1connect.QuizServiceListenForEventsProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
	assert.False(t, asked, "the stream asks planet nothing")
}

func TestNotFrozenAQuizIsAnswered(t *testing.T) {
	ran, err := call(t, always(false), quizzesv1connect.QuizServiceAnswerQuizProcedure)

	require.NoError(t, err)
	assert.True(t, ran)
}

func TestAFreezeNobodyCanTellTakesNothing(t *testing.T) {
	ran, err := call(t, func(context.Context) (bool, error) { return false, errors.New("planet is gone") },
		quizzesv1connect.QuizServiceAnswerQuizProcedure)

	require.Error(t, err)
	assert.False(t, ran)
}
