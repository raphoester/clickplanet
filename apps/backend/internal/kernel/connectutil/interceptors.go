package connectutil

import (
	"context"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

type Limiter interface {
	Allow(key string) bool
}

type Blocklist interface {
	Blocked(ip string) (ipblock.List, bool)
}

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

// SessionHeader carries the token minted by session.v1.SessionService. A custom
// header rather than a field on each request: it is an edge concern, the same
// on every procedure, and keeping it out of the message means the contract of
// the game does not change to describe how a caller is admitted to it.
//
// It is also why the CORS allowlist has to name it — a custom header on a
// cross-origin POST is what turns that POST into a preflight.
const SessionHeader = "X-Session-Token"

type SessionVerifier interface {
	Verify(token string, ip string, now time.Time) (session.ID, error)
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
	timeProvider xtime.Provider,
	refusal error,
	enforce bool,
	onVerdict func(SessionVerdict),
	procedures ...string,
) connect.Interceptor {
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
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

			id, err := verifier.Verify(token, ctxutil.GetSourceIP(ctx), timeProvider.Now())
			if err != nil {
				record(SessionInvalid)
				if enforce {
					return nil, connect.NewError(connect.CodeUnauthenticated, refusal)
				}
				return next(ctx, req)
			}

			record(SessionValid)

			return next(ctxutil.AddSessionIDToContext(ctx, string(id)), req)
		}
	})
}
