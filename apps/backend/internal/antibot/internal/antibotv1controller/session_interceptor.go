package antibotv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("the antibot service requires a session; call auth.v1.AuthService/CreateSession first")

func NewSessionInterceptor(verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "antibot_session_checks",
		Help: "Antibot service calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Identified(antibotv1connect.AntibotServiceAcknowledgeWarningProcedure),
	)
}

func NewStreamSessionReader(verifier cpconnect.SessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(antibotv1connect.AntibotServiceListenForEventsProcedure))
}
