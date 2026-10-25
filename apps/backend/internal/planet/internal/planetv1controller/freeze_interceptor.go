package planetv1controller

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

var ErrMapFrozen = errors.New("the map is frozen: nobody plays on it until the rules in force open it again")

type MapRules interface {
	Frozen() bool
}

var writes = []string{
	planetv1connect.ClickServiceClickProcedure,
	planetv1connect.ClickServiceDropBombProcedure,
	planetv1connect.ClickServiceUseRefillProcedure,
	planetv1connect.ClickServicePlaceShieldProcedure,
	planetv1connect.ClickServiceClaimBonusProcedure,
	planetv1connect.ClickServiceAnswerQuizProcedure,
}

func NewFreezeInterceptor(rules MapRules, registerer prometheus.Registerer) connect.Interceptor {
	frozen := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "frozen_clicks",
		Help: "Clicks refused because the rules in force freeze the map",
	})

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			if !slices.Contains(writes, procedure) || !rules.Frozen() {
				return next(ctx, req)
			}

			if procedure == planetv1connect.ClickServiceClickProcedure {
				frozen.Inc()
			}

			refused := connect.NewError(connect.CodeFailedPrecondition, ErrMapFrozen)
			if detail, err := connect.NewErrorDetail(&planetv1.MapFrozen{}); err == nil {
				refused.AddDetail(detail)
			}
			return nil, refused
		}
	})
}
