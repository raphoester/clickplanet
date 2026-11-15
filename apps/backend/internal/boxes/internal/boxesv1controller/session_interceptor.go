package boxesv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("boxes require a session; call auth.v1.AuthService/CreateSession first")

// NewSessionInterceptor asks a claim for the click token a click carries, as planet asked it before the boxes left it.// module of their own.
func NewSessionInterceptor(
	verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_session_checks",
		Help: "Box claims by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(boxesv1connect.BoxServiceClaimBoxProcedure),
	)
}

// NewStreamSessionReader reads the token a stream opens with, and refuses nothing: a signed-in account's stream attends
// the boxes as that account, any other as its address.
func NewStreamSessionReader(verifier cpconnect.SessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(boxesv1connect.BoxServiceListenForEventsProcedure))
}
