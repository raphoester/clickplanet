package planetv1controller

import (
	"context"
	"slices"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The calls that change the map or a player's play. Reads and the stream go on while the map is frozen.
var playProcedures = []string{
	planetv1connect.ClickServiceClickProcedure,
	planetv1connect.ClickServiceDropBombProcedure,
	planetv1connect.ClickServicePlaceShieldProcedure,
	planetv1connect.ClickServiceUseRefillProcedure,
	planetv1connect.ClickServiceClaimBonusProcedure,
	planetv1connect.ClickServiceOpenQuizProcedure,
	planetv1connect.ClickServiceAnswerQuizProcedure,
}

// NewFreezeInterceptor refuses every play while the rules in force freeze the map, before a call spends anything.
func NewFreezeInterceptor(freeze func() clicks.Freeze) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !bool(freeze()) || !slices.Contains(playProcedures, req.Spec().Procedure) {
				return next(ctx, req)
			}
			refused := connect.NewError(connect.CodeFailedPrecondition, clicks.ErrFrozen)
			if detail, err := connect.NewErrorDetail(&planetv1.FrozenRefusal{}); err == nil {
				refused.AddDetail(detail)
			}
			return nil, refused
		}
	})
}
