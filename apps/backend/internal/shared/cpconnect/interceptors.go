package cpconnect

import (
	"context"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"google.golang.org/protobuf/proto"
)

type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

type Blocklist interface {
	Blocked(ip string) (cpipblock.List, bool)
}

// RateLimitKey is the identity a bucket is kept under.
//
// Keyed on the scope rather than the address: an IPv6 caller owns every address
// in its own /64, so a bucket per address is one it steps out of for free. See
// cpipscope. Anything that reports an allowance must derive the key the same way,
// or it reports somebody else's.
func RateLimitKey(ctx context.Context) string {
	return cpipscope.Of(cpctx.GetSourceIP(ctx))
}

// NewRateLimitInterceptor spends a token per call on the named procedures.
//
// What the bucket has left travels onward both ways: onto the context when the
// call passes, so the handler can put it in its answer, and — where describe is
// given — onto the refusal as an error detail, since a refused call has no
// answer to carry it. A client that shows the allowance to a player therefore
// learns it from its own calls and never has to poll for it.
func NewRateLimitInterceptor(
	limiter Limiter,
	refusal error,
	describe func(cpratelimit.State) proto.Message,
	procedures ...string,
) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			allowed, state := limiter.Take(RateLimitKey(ctx))
			if !allowed {
				return nil, refuse(refusal, describe, state)
			}

			return next(cpctx.AddRateBudgetToContext(ctx, state), req)
		}
	})
}

func refuse(refusal error, describe func(cpratelimit.State) proto.Message, state cpratelimit.State) error {
	err := connect.NewError(connect.CodeResourceExhausted, refusal)
	if describe == nil {
		return err
	}

	// Only a message that will not marshal fails here, which a generated one
	// does not. The caller still has to be refused either way, so it is the
	// detail that is dropped rather than the refusal that becomes an error.
	if detail, detailErr := connect.NewErrorDetail(describe(state)); detailErr == nil {
		err.AddDetail(detail)
	}

	return err
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
	Verify(token string, ip string, now time.Time) (cpsession.ID, error)
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

			id, err := verifier.Verify(token, cpctx.GetSourceIP(ctx), clock.Now())
			if err != nil {
				record(SessionInvalid)
				if enforce {
					return nil, connect.NewError(connect.CodeUnauthenticated, refusal)
				}
				return next(ctx, req)
			}

			record(SessionValid)

			return next(cpctx.AddSessionIDToContext(ctx, string(id)), req)
		}
	})
}
