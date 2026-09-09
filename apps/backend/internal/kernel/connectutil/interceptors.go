// Package connectutil holds the Connect interceptors both bounded contexts
// need. They are here rather than in either context because the policy they
// enforce — who may call, and how often — is the same policy whatever the
// procedure is; only the procedure names and the wording of the refusal differ,
// and those are arguments.
package connectutil

import (
	"context"
	"slices"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
)

// Limiter answers whether a key — a source IP here — may act now.
// kernel/ratelimit implements it.
type Limiter interface {
	Allow(key string) bool
}

// Blocklist answers whether an address is refused, and which list said so.
// *ipblock.Blocklist implements it, whether it was built from the vendored
// lists or from an operator's deny list.
type Blocklist interface {
	Blocked(ip string) (ipblock.List, bool)
}

// NewRateLimitInterceptor throttles the named procedures per source IP, from
// the caller's bucket. Procedures it was not given are passed straight through:
// a cacheable read is absorbed by the proxy in front, and limiting one would
// punish a page load rather than a bot.
//
// The key is whatever IPReaderMiddleware put on the context. A refused call
// answers refusal with CodeResourceExhausted, i.e. HTTP 429, and never reaches
// the domain.
func NewRateLimitInterceptor(limiter Limiter, refusal error, procedures ...string) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			if !limiter.Allow(ctxutil.GetSourceIP(ctx)) {
				return nil, connect.NewError(connect.CodeResourceExhausted, refusal)
			}

			return next(ctx, req)
		}
	})
}

// NewIPBlockInterceptor refuses the named procedures from a blocked address
// with CodePermissionDenied, i.e. HTTP 403. onBlocked, when non-nil, is called
// with the list that matched — the hook the click counter hangs on.
//
// An address the middleware could not resolve is passed through rather than
// refused: this list exists to refuse the known-bad, not to refuse everything
// it cannot read.
func NewIPBlockInterceptor(
	blocklist Blocklist,
	refusal error,
	onBlocked func(ipblock.List),
	procedures ...string,
) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			ip := ctxutil.GetSourceIP(ctx)
			if ip == "" {
				return next(ctx, req)
			}

			list, isBlocked := blocklist.Blocked(ip)
			if !isBlocked {
				return next(ctx, req)
			}

			if onBlocked != nil {
				onBlocked(list)
			}

			return nil, connect.NewError(connect.CodePermissionDenied, refusal)
		}
	})
}
