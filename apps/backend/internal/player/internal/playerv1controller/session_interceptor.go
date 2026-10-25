package playerv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("the player service requires a session; call auth.v1.AuthService/CreateSession first")

func NewSessionInterceptor(verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "player_session_checks",
		Help: "Player service calls by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		true,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Identified(playerv1connect.PlayerServiceGetProfileProcedure),
		cpconnect.Attested(playerv1connect.PlayerServiceSetNameProcedure),
		cpconnect.Attested(playerv1connect.PlayerServiceSetColorProcedure),
		cpconnect.Identified(playerv1connect.PlayerServiceGetStatsProcedure),
		cpconnect.Attested(playerv1connect.PlayerServiceAnnounceProcedure),
		cpconnect.Identified(playerv1connect.PlayerServiceLeaveProcedure),
		cpconnect.Identified(playerv1connect.PlayerServiceGetTitlesProcedure),
		cpconnect.Attested(playerv1connect.PlayerServiceWearTitleProcedure),
		cpconnect.Identified(playerv1connect.PlayerServiceGetFrontsProcedure),
	)
}

func NewStreamSessionReader(verifier cpconnect.SessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(playerv1connect.PlayerServiceListenForEventsProcedure))
}
