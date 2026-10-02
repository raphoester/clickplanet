package planetv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

// mapMaxAge is how long a proxy may serve a map read from cache. A burst of
// visitors can then share one origin response: ListenForEvents carries
// everything that happened after a chunk was built, so a client starting from a
// slightly old map converges anyway.
const mapMaxAge = 5

// rulesMaxAge is how long GetBonusRules may be served from cache. The rules only change with a deploy, and a
// client reads them once per page load, so a stale answer lasts a few minutes after a deploy at worst.
const rulesMaxAge = 300

// NewCacheInterceptor marks the three cacheable reads, and nothing else.
//
// It is one interceptor rather than a header set by each handler because the
// policy is a property of the procedure, not of the answer: GetMap,
// MapDensity and GetBonusRules are the three marked NO_SIDE_EFFECTS in the proto, so Connect sends
// them as GETs, and this is the list that has to agree with that one. GetBudget
// is deliberately absent — it is a POST precisely so no cache serves a stale
// allowance, and GetHistory in chat answers no-store for the same reason.
func NewCacheInterceptor() connect.Interceptor {
	maxAge := map[string]int{
		planetv1connect.ClickServiceGetMapProcedure:        mapMaxAge,
		planetv1connect.ClickServiceMapDensityProcedure:    mapMaxAge,
		planetv1connect.ClickServiceGetBonusRulesProcedure: rulesMaxAge,
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
