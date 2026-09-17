package cpconnect

import (
	"context"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

type Blocklist interface {
	Blocked(ip string) (cpipblock.List, bool)
}

// NewRateLimitInterceptor spends a token per call on the named procedures, and
// answers CodeResourceExhausted when there is none.
//
// It reports nothing about what is left. A context that shows a player their
// allowance throttles inside its own use case instead, where the reading is a
// return value rather than something smuggled along the context — see the
// clicks module. This is for the procedures where a refusal is the whole story.
func NewRateLimitInterceptor(
	limiter Limiter,
	refusal error,
	procedures ...string,
) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			if allowed, _ := limiter.Take(cpctx.RateLimitKey(ctx)); !allowed {
				return nil, connect.NewError(connect.CodeResourceExhausted, refusal)
			}

			return next(ctx, req)
		}
	})
}

func NewIPBlockInterceptor(
	blocklist Blocklist,
	refusal error,
	onBlocked func(cpipblock.List),
	procedures ...string,
) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			ip := cpctx.GetSourceIP(ctx)
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

// SessionHeader carries the token minted by cpsession.v1.SessionService. A custom
// header rather than a field on each request: it is an edge concern, the same
// on every procedure, and keeping it out of the message means the contract of
// the game does not change to describe how a caller is admitted to it.
//
// It is also why the CORS allowlist has to name it — a custom header on a
// cross-origin POST is what turns that POST into a preflight.
const SessionHeader = "X-Session-Token"

type SessionVerifier interface {
	Verify(token string, ip string, now time.Time) (*cpsession.Claims, error)
}

// SessionVerdict labels what happened, so the counter can show what enforcing
// would refuse before it is enforced.
type SessionVerdict string

const (
	SessionValid   SessionVerdict = "valid"
	SessionMissing SessionVerdict = "missing"
	SessionInvalid SessionVerdict = "invalid"
)

// NewSessionInterceptor requires a minted session on the given procedures.
//
// With enforce false it decides nothing and only counts: every caller is passed
// through with its verdict recorded, which is how this ships in front of clients
// that do not send a token yet. With enforce true a caller without a valid one
// is answered CodeUnauthenticated, and the client is expected to mint and retry
// rather than to show the player an error.
func NewSessionInterceptor(
	verifier SessionVerifier,
	clock cptime.Clock,
	refusal error,
	enforce bool,
	onVerdict func(SessionVerdict),
	procedures ...string,
) connect.Interceptor {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	record := func(verdict SessionVerdict) {
		if onVerdict != nil {
			onVerdict(verdict)
		}
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			token := req.Header().Get(SessionHeader)
			if token == "" {
				record(SessionMissing)
				if enforce {
					return nil, connect.NewError(connect.CodeUnauthenticated, refusal)
				}
				return next(ctx, req)
			}

			claims, err := verifier.Verify(token, cpctx.GetSourceIP(ctx), clock.Now())
			if err != nil {
				record(SessionInvalid)
				if enforce {
					return nil, connect.NewError(connect.CodeUnauthenticated, refusal)
				}
				return next(ctx, req)
			}

			record(SessionValid)

			return next(withClaims(ctx, claims), req)
		}
	})
}

// NewSessionReaderInterceptor reads a token on the given procedures when one is sent, and refuses nothing.
// It is for a read that answers better for an account, such as the click budget.
func NewSessionReaderInterceptor(verifier SessionVerifier, clock cptime.Clock, procedures ...string) connect.Interceptor {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := req.Header().Get(SessionHeader)
			if token == "" || !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			claims, err := verifier.Verify(token, cpctx.GetSourceIP(ctx), clock.Now())
			if err != nil {
				return next(ctx, req)
			}

			return next(withClaims(ctx, claims), req)
		}
	})
}

// withClaims puts the session id on the context, and the account when the token names one.
func withClaims(ctx context.Context, claims *cpsession.Claims) context.Context {
	ctx = cpctx.AddSessionIDToContext(ctx, string(claims.ID))
	if claims.Account == cpsession.NoAccount {
		return ctx
	}

	return cpctx.AddAccountToContext(ctx, claims.Account.String())
}
