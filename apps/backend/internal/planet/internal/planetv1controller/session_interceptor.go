package planetv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("clicks require a session; call session.v1.SessionService/CreateSession first")

type ClickSessionVerifier = cpconnect.SessionVerifier

func NewSessionInterceptor(
	verifier ClickSessionVerifier,
	clock cptime.Clock,
	enforce bool,
	registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "click_session_checks",
		Help: "Clicks by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		enforce,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(planetv1connect.ClickServiceClickProcedure),
		cpconnect.Attested(planetv1connect.ClickServiceClaimBonusProcedure),
		cpconnect.Attested(planetv1connect.ClickServiceDropBombProcedure),
		cpconnect.Attested(planetv1connect.ClickServiceUseRefillProcedure),
		cpconnect.Attested(planetv1connect.ClickServiceOpenQuizProcedure),
		cpconnect.Attested(planetv1connect.ClickServiceAnswerQuizProcedure),
	)
}

func NewSessionReaderInterceptor(verifier ClickSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(planetv1connect.ClickServiceGetBudgetProcedure),
		cpconnect.Identified(planetv1connect.ClickServiceGetChargesProcedure),
		cpconnect.Identified(planetv1connect.ClickServiceListenForEventsProcedure),
	)
}
