package promov1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
)

// A teaser is read as a page opens: a card or a rule changed shows within this long.
const teaserMaxAge = 60

func NewCacheInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			if err != nil || req.Spec().Procedure != promov1connect.PromoServiceGetTeaserProcedure {
				return res, err
			}
			res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", teaserMaxAge, teaserMaxAge))
			return res, nil
		}
	})
}
