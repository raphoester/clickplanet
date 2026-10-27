package scoringv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
)

const (
	seasonMaxAge    = 60
	standingsMaxAge = 15
	// The race changes once a minute, with the snapshot.
	raceMaxAge       = 10
	hallOfFameMaxAge = 60
)

func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		scoringv1connect.SeasonServiceGetSeasonProcedure:     seasonMaxAge,
		scoringv1connect.SeasonServiceGetStandingsProcedure:  standingsMaxAge,
		scoringv1connect.SeasonServiceGetRaceProcedure:       raceMaxAge,
		scoringv1connect.SeasonServiceGetHallOfFameProcedure: hallOfFameMaxAge,
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			age, cacheable := maxAge[req.Spec().Procedure]
			if err != nil || !cacheable {
				return res, err
			}

			res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", age))

			return res, nil
		}
	})
}
