package scoringv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
)

const (
	boardMaxAge = 15
	// A round's standings and a book's scores change once a round closes, and as the map moves.
	roundMaxAge  = 10
	scoresMaxAge = 10
)

func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		scoringv1connect.ScoringServiceGetBoardProcedure:       boardMaxAge,
		scoringv1connect.ScoringServiceGetRoundProcedure:       roundMaxAge,
		scoringv1connect.ScoringServiceGetScoresProcedure:      scoresMaxAge,
		scoringv1connect.ScoringServiceGetBookHistoryProcedure: scoresMaxAge,
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
