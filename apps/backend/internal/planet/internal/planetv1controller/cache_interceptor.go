package planetv1controller

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

// mapMaxAge is how long a proxy may serve a map read from cache. A burst of
// visitors can then share one origin response: ListenForEvents carries
// everything that happened after a chunk was built, so a client starting from a
// slightly old map converges anyway.
const mapMaxAge = 5

// NewCacheInterceptor marks the two cacheable reads, and nothing else.
//
// It is one interceptor rather than a header set by each handler because the
// policy is a property of the procedure, not of the answer: GetMap and
// MapDensity are the two marked NO_SIDE_EFFECTS in the proto, so Connect sends
// them as GETs, and this is the list that has to agree with that one. GetBudget
// is deliberately absent — it is a POST precisely so no cache serves a stale
// allowance, and GetHistory in chat answers no-store for the same reason.
func NewCacheInterceptor() connect.Interceptor {
	cacheable := []string{
		planetv1connect.ClickServiceGetMapProcedure,
		planetv1connect.ClickServiceMapDensityProcedure,
	}

	value := fmt.Sprintf("public, max-age=%d", mapMaxAge)

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			if err != nil || !slices.Contains(cacheable, req.Spec().Procedure) {
				return res, err
			}

			res.Header().Set("Cache-Control", value)

			return res, nil
		}
	})
}
