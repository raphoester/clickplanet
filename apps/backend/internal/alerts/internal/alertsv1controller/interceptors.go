package alertsv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1/alertsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoSession    = errors.New("the alerts service requires a session; call auth.v1.AuthService/CreateSession first")
	ErrTooManyTries = errors.New("too many Discord links started")
)

// Each one writes nothing but its caller's own link.
func NewSessionInterceptor(verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "alerts_session_checks",
		Help: "Alerts service calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		true,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Identified(alertsv1connect.AlertsServiceGetDiscordLinkProcedure),
		cpconnect.Identified(alertsv1connect.AlertsServiceStartDiscordLinkProcedure),
		cpconnect.Identified(alertsv1connect.AlertsServiceCompleteDiscordLinkProcedure),
	)
}

// Each trip to Discord costs a call to it.
func NewRateLimitInterceptor(limiter cpconnect.Limiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManyTries,
		alertsv1connect.AlertsServiceStartDiscordLinkProcedure,
		alertsv1connect.AlertsServiceCompleteDiscordLinkProcedure,
	)
}
