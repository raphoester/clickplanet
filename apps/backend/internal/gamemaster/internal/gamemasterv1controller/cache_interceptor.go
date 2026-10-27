package gamemasterv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
)

// The client reads the turn again at the entry's end: a cached answer lies for at most this long.
const turnMaxAge = 5

func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		gamemasterv1connect.GamemasterServiceGetTurnProcedure: turnMaxAge,
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			age, cacheable := maxAge[req.Spec().Procedure]
			if err != nil || !cacheable {
				return res, err
			}

			res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", age, age))

			return res, nil
		}
	})
}
