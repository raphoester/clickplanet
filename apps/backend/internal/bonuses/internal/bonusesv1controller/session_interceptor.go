package bonusesv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("boxes and quizzes require a session; call auth.v1.AuthService/CreateSession first")

// NewSessionInterceptor asks a claim, an opening and an answer for the click token a click carries, as planet asked
// them before the offers were a module of their own.
func NewSessionInterceptor(
	verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "bonus_session_checks",
		Help: "Box claims, quiz openings and answers by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(bonusesv1connect.BonusServiceClaimBonusProcedure),
		cpconnect.Attested(bonusesv1connect.BonusServiceOpenQuizProcedure),
		cpconnect.Attested(bonusesv1connect.BonusServiceAnswerQuizProcedure),
	)
}

// NewStreamSessionReader reads the token a stream opens with, and refuses nothing: a signed-in account's stream attends
// the offers as that account, any other as its address.
func NewStreamSessionReader(verifier cpconnect.SessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(bonusesv1connect.BonusServiceListenForEventsProcedure))
}
