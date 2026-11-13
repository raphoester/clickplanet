package quizzesv1controller

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
)

// The calls that take a quiz. The stream goes on while frozen.
var takeProcedures = []string{
	quizzesv1connect.QuizServiceOpenQuizProcedure,
	quizzesv1connect.QuizServiceAnswerQuizProcedure,
}

// Frozen says whether a quiz may be taken now: no while the quizzes play a frozen set, nor while planet's map takes no
// play (its frozen set, an arena's freeze).
type Frozen func(ctx context.Context) (bool, error)

// NewFreezeInterceptor refuses an opening and an answer while frozen, before the quiz is spent: it lapses as it would
// have.
func NewFreezeInterceptor(frozen Frozen) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(takeProcedures, req.Spec().Procedure) {
				return next(ctx, req)
			}
			isFrozen, err := frozen(ctx)
			if err != nil {
				return nil, err
			}
			if !isFrozen {
				return next(ctx, req)
			}
			refused := connect.NewError(connect.CodeFailedPrecondition, quizzes.ErrFrozen)
			if detail, err := connect.NewErrorDetail(&quizzesv1.FrozenRefusal{}); err == nil {
				refused.AddDetail(detail)
			}
			return nil, refused
		}
	})
}
