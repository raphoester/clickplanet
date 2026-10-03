package marketingv1controller

import (
	"context"
	"crypto/subtle"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/marketing/v1/marketingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoSession = errors.New("season emails need a session; call auth.v1.AuthService/CreateSession first")
	ErrThrottled = errors.New("too many changes to season emails lately; wait a minute")
	ErrNotBrevo  = errors.New("this call does not carry the webhook secret")
)

var writes = []string{
	marketingv1connect.SubscriptionServiceSubscribeProcedure,
	marketingv1connect.SubscriptionServiceUnsubscribeProcedure,
}

func NewSessionInterceptor(verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "marketing_session_checks",
		Help: "Season email calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		true,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		marketingv1connect.SubscriptionServiceGetSubscriptionProcedure,
		marketingv1connect.SubscriptionServiceSubscribeProcedure,
		marketingv1connect.SubscriptionServiceUnsubscribeProcedure,
	)
}

// Inside the session interceptor: the bucket is the account it put on the context.
func NewRateLimitInterceptor(limiter cpconnect.Limiter) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			account := cpctx.GetAccount(ctx)
			if account == "" || !slices.Contains(writes, req.Spec().Procedure) {
				return next(ctx, req)
			}
			if allowed, _ := limiter.Take(account); !allowed {
				return nil, connect.NewError(connect.CodeResourceExhausted, ErrThrottled)
			}
			return next(ctx, req)
		}
	})
}

func NewWebhookInterceptor(secret string) connect.Interceptor {
	want := []byte("Bearer " + secret)

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			got := []byte(req.Header().Get("Authorization"))
			if secret == "" || subtle.ConstantTimeCompare(got, want) != 1 {
				return nil, connect.NewError(connect.CodeUnauthenticated, ErrNotBrevo)
			}
			return next(ctx, req)
		}
	})
}
