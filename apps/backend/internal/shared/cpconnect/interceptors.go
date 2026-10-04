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

const SessionHeader = "X-Session-Token"

type SessionVerifier interface {
	Verify(ctx context.Context, token string, ip string, now time.Time) (*cpsession.Claims, error)
}

type SessionVerdict string

const (
	SessionValid      SessionVerdict = "valid"
	SessionMissing    SessionVerdict = "missing"
	SessionInvalid    SessionVerdict = "invalid"
	SessionUnattested SessionVerdict = "unattested"
)

// A procedure that reads its caller says which token it takes, so none takes the weaker one by default.
type Procedure struct {
	name       string
	identified bool
}

// Attested acts for its caller: only a token minted after a Turnstile check names it.
func Attested(name string) Procedure {
	return Procedure{name: name}
}

// Identified only reads, or writes nothing but its caller's own state: any token auth minted names it.
func Identified(name string) Procedure {
	return Procedure{name: name, identified: true}
}

func (p Procedure) takes(claims *cpsession.Claims) bool {
	return p.identified || claims.Attested
}

func procedureOf(procedures []Procedure, name string) (Procedure, bool) {
	for _, procedure := range procedures {
		if procedure.name == name {
			return procedure, true
		}
	}
	return Procedure{}, false
}

func NewSessionInterceptor(
	verifier SessionVerifier,
	clock cptime.Clock,
	refusal error,
	enforce bool,
	onVerdict func(SessionVerdict),
	procedures ...Procedure,
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
			procedure, listed := procedureOf(procedures, req.Spec().Procedure)
			if !listed {
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

			claims, err := verifier.Verify(ctx, token, cpctx.GetSourceIP(ctx), clock.Now())
			if err != nil {
				record(SessionInvalid)
				if enforce {
					return nil, connect.NewError(connect.CodeUnauthenticated, refusal)
				}
				return next(ctx, req)
			}

			if !procedure.takes(claims) {
				record(SessionUnattested)
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

func NewSessionReaderInterceptor(verifier SessionVerifier, clock cptime.Clock, procedures ...Procedure) connect.Interceptor {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return sessionReader{verifier: verifier, clock: clock, procedures: procedures}
}

type sessionReader struct {
	verifier   SessionVerifier
	clock      cptime.Clock
	procedures []Procedure
}

func (r sessionReader) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return next(r.context(ctx, req.Spec().Procedure, req.Header().Get(SessionHeader)), req)
	}
}

func (r sessionReader) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (r sessionReader) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(r.context(ctx, conn.Spec().Procedure, conn.RequestHeader().Get(SessionHeader)), conn)
	}
}

func (r sessionReader) context(ctx context.Context, name string, token string) context.Context {
	procedure, listed := procedureOf(r.procedures, name)
	if token == "" || !listed {
		return ctx
	}

	claims, err := r.verifier.Verify(ctx, token, cpctx.GetSourceIP(ctx), r.clock.Now())
	if err != nil || !procedure.takes(claims) {
		return ctx
	}

	return withClaims(ctx, claims)
}

func withClaims(ctx context.Context, claims *cpsession.Claims) context.Context {
	ctx = cpctx.AddSessionIDToContext(ctx, string(claims.ID))
	if claims.Account == cpsession.NoAccount {
		return ctx
	}

	ctx = cpctx.AddAccountToContext(ctx, claims.Account.String())
	if created, ok := claims.Account.CreatedAt(); ok {
		ctx = cpctx.AddAccountCreatedToContext(ctx, created)
	}
	if !claims.Linked {
		return ctx
	}

	return cpctx.AddLinkedToContext(ctx)
}
