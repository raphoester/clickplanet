package planetv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

const mapMaxAge = 5

const rulesMaxAge = 300

func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		planetv1connect.ClickServiceGetMapProcedure:        mapMaxAge,
		planetv1connect.ClickServiceMapDensityProcedure:    mapMaxAge,
		planetv1connect.ClickServiceGetBonusRulesProcedure: rulesMaxAge,
		planetv1connect.ClickServiceGetFortressesProcedure: mapMaxAge,
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
