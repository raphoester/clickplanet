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

// ErrNoSession is the answer to a call with no valid click token: the client mints one and retries.
var ErrNoSession = errors.New("the player service requires a session; call auth.v1.AuthService/CreateSession first")

// NewSessionInterceptor refuses every PlayerService call but GetRoster, GetPlayer and ListenForEvents without a
// valid click token. It always enforces: a profile is an account's, and there is no caller to answer for without
// one. Those three answer the same to anybody.
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
		playerv1connect.PlayerServiceGetProfileProcedure,
		playerv1connect.PlayerServiceSetNameProcedure,
		playerv1connect.PlayerServiceGetStatsProcedure,
		playerv1connect.PlayerServiceAnnounceProcedure,
		playerv1connect.PlayerServiceLeaveProcedure,
	)
}
