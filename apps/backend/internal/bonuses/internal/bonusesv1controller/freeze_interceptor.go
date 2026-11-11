package bonusesv1controller

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// The calls that take a box or a quiz. The stream and the reads go on while frozen.
var takeProcedures = []string{
	bonusesv1connect.BonusServiceClaimBonusProcedure,
	bonusesv1connect.BonusServiceOpenQuizProcedure,
	bonusesv1connect.BonusServiceAnswerQuizProcedure,
}

// Frozen says whether a box or a quiz may be taken now: no while the offers play a frozen set, nor while planet's map
// takes no play (its frozen set, an arena's freeze).
type Frozen func(ctx context.Context) (bool, error)

// NewFreezeInterceptor refuses a claim, an opening and an answer while frozen, before the offer is spent: the box or
// the quiz lapses as it would have.
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
			refused := connect.NewError(connect.CodeFailedPrecondition, bonuses.ErrFrozen)
			if detail, err := connect.NewErrorDetail(&bonusesv1.FrozenRefusal{}); err == nil {
				refused.AddDetail(detail)
			}
			return nil, refused
		}
	})
}
