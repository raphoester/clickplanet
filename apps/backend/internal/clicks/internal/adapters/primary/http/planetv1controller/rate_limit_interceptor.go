package planetv1controller

import (
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"google.golang.org/protobuf/proto"
)

type ClickLimiter = connectutil.Limiter

func NewRateLimitInterceptor(limiter ClickLimiter) connect.Interceptor {
	return connectutil.NewRateLimitInterceptor(
		limiter,
		errors.New("too many clicks"),
		describeBudget,
		planetv1connect.ClickServiceClickProcedure,
	)
}

// describeBudget is what a refused click carries back, so the counter on screen
// drops to zero at the same moment the server does, and starts its wait from
// the server's own reading rather than from a guess.
func describeBudget(state ratelimit.State) proto.Message {
	return EncodeBudget(state)
}

// EncodeBudget puts a limiter reading on the wire. Capacity and refill rate go
// with it: the client redraws the allowance many times a second, and it can
// only do that without asking again if it knows the policy it must replay.
func EncodeBudget(state ratelimit.State) *planetv1.ClickBudget {
	return &planetv1.ClickBudget{
		Tokens:          max(state.Tokens, 0),
		Capacity:        uint32(state.Capacity),
		RefillPerSecond: state.PerSecond,
	}
}
