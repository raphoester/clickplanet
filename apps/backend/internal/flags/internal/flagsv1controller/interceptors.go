package flagsv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoSession      = errors.New("a flag's submission requires a session; call auth.v1.AuthService/CreateSession first")
	ErrTooManyUploads = errors.New("too many pictures of flags uploaded today")
)

func NewSessionInterceptor(
	verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "flags_session_checks",
		Help: "Flag submission calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(flagsv1connect.FlagServiceCreateSubmissionUploadProcedure),
		cpconnect.Attested(flagsv1connect.FlagServiceSubmitFlagProcedure),
		cpconnect.Attested(flagsv1connect.FlagServiceAppealSubmissionProcedure),
		cpconnect.Identified(flagsv1connect.FlagServiceListMySubmissionsProcedure),
	)
}

type DailyLimiter interface {
	TakeAll(n float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State)
}

// A day's pictures, for the account and for its address together: each one is room in the bucket. Inside the session
// check.
func NewUploadLimitInterceptor(limiter DailyLimiter) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().Procedure != flagsv1connect.FlagServiceCreateSubmissionUploadProcedure {
				return next(ctx, req)
			}
			keys := []cpratelimit.Key{{Name: "scope:" + cpctx.RateLimitKey(ctx)}}
			if account := cpctx.GetAccount(ctx); account != "" {
				keys = append(keys, cpratelimit.Key{Name: "account:" + account})
			}
			if allowed, _ := limiter.TakeAll(1, keys...); !allowed {
				return nil, connect.NewError(connect.CodeResourceExhausted, ErrTooManyUploads)
			}
			return next(ctx, req)
		}
	})
}
