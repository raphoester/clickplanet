package cpctx

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

// The scope, not the address: an IPv6 caller owns every address in its /64.
func RateLimitKey(ctx context.Context) string {
	return cpipscope.Of(GetSourceIP(ctx))
}
