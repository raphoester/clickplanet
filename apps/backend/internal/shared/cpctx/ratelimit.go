package cpctx

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

// RateLimitKey is the identity a bucket is kept under.
//
// Keyed on the scope rather than the address: an IPv6 caller owns every address
// in its own /64, so a bucket per address is one it steps out of for free. See
// cpipscope. Anything that spends or reports an allowance derives it here, or it
// charges one caller and reports another.
func RateLimitKey(ctx context.Context) string {
	return cpipscope.Of(GetSourceIP(ctx))
}
