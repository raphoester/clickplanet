package supportv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoSession   = errors.New("the support service requires a session; call auth.v1.AuthService/CreateSession first")
	ErrTooManySent = errors.New("too many sent to the support service")
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
	)
}

func NewRateLimitInterceptor(limiter cpconnect.Limiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManySent,
		supportv1connect.SupportServiceSuggestFeatureProcedure,
		supportv1connect.SupportServiceReportBugProcedure,
		supportv1connect.SupportServiceReportPlayerProcedure,
	)
}
