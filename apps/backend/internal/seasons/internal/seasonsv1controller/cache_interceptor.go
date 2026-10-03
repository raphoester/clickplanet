package seasonsv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
)

const seasonMaxAge = 60

func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		seasonsv1connect.SeasonServiceGetSeasonProcedure: seasonMaxAge,
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
