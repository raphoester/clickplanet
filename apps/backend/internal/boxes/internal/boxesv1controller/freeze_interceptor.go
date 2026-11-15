package boxesv1controller

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
)

// The calls that take a box. The stream and the reads go on while frozen.
var takeProcedures = []string{
	boxesv1connect.BoxServiceClaimBoxProcedure,
}

// Frozen says whether a box may be taken now: no while the boxes play a frozen set, nor while planet's map
// takes no play (its frozen set, an arena's freeze).
type Frozen func(ctx context.Context) (bool, error)

// NewFreezeInterceptor refuses a claim while frozen, before the offer is spent: the box lapses as it would have.
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
			refused := connect.NewError(connect.CodeFailedPrecondition, boxes.ErrFrozen)
			if detail, err := connect.NewErrorDetail(&boxesv1.FrozenRefusal{}); err == nil {
				refused.AddDetail(detail)
			}
			return nil, refused
		}
	})
}
