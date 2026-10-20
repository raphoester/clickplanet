package supportv1controller

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoSession   = errors.New("the support service requires a session; call auth.v1.AuthService/CreateSession first")
	ErrTooManySent = errors.New("too many sent to the support service")

	ErrTooManyNotes   = errors.New("too many notes sent to the support service today")
	ErrTooManyUploads = errors.New("too many files uploaded to the support service today")
)

func NewSessionInterceptor(verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "support_session_checks",
		Help: "Support service calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		true,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(supportv1connect.SupportServiceSuggestFeatureProcedure),
		cpconnect.Attested(supportv1connect.SupportServiceReportBugProcedure),
		cpconnect.Attested(supportv1connect.SupportServiceReportPlayerProcedure),
		cpconnect.Attested(supportv1connect.SupportServiceReportPlaceProcedure),
		cpconnect.Attested(supportv1connect.SupportServiceCreateUploadProcedure),
	)
}

func NewRateLimitInterceptor(limiter cpconnect.Limiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManySent,
		supportv1connect.SupportServiceSuggestFeatureProcedure,
		supportv1connect.SupportServiceReportBugProcedure,
		supportv1connect.SupportServiceReportPlayerProcedure,
		supportv1connect.SupportServiceReportPlaceProcedure,
	)
}

type DailyLimiter interface {
	TakeAll(n float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State)
}

// A day's suggestions and bug reports, for the account and for its address.
func NewNoteLimitInterceptor(limiter DailyLimiter) connect.Interceptor {
	return newDailyLimitInterceptor(limiter, ErrTooManyNotes,
		supportv1connect.SupportServiceSuggestFeatureProcedure,
		supportv1connect.SupportServiceReportBugProcedure,
	)
}

// A day's files, for the account and for its address: each one is room in the bucket.
func NewUploadLimitInterceptor(limiter DailyLimiter) connect.Interceptor {
	return newDailyLimitInterceptor(limiter, ErrTooManyUploads, supportv1connect.SupportServiceCreateUploadProcedure)
}

// Spends from the caller's account and from its address together, or from neither. Inside the session check.
func newDailyLimitInterceptor(limiter DailyLimiter, refusal error, procedures ...string) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !slices.Contains(procedures, req.Spec().Procedure) {
				return next(ctx, req)
			}

			keys := []cpratelimit.Key{{Name: "scope:" + cpctx.RateLimitKey(ctx)}}
			if account := cpctx.GetAccount(ctx); account != "" {
				keys = append(keys, cpratelimit.Key{Name: "account:" + account})
			}
			if allowed, _ := limiter.TakeAll(1, keys...); !allowed {
				return nil, connect.NewError(connect.CodeResourceExhausted, refusal)
			}

			return next(ctx, req)
		}
	})
}
