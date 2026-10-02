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

// ErrNoSession is a refusal the player never reads: the client mints a token
// and retries the click. It is only visible to something that is not the
// client, which is the point.
var ErrNoSession = errors.New("clicks require a session; call session.v1.SessionService/CreateSession first")

type ClickSessionVerifier = cpconnect.SessionVerifier

func NewSessionInterceptor(
	verifier ClickSessionVerifier,
	clock cptime.Clock,
	enforce bool,
	registerer prometheus.Registerer,
) connect.Interceptor {
	// Labelled per verdict so the cost of enforcing is visible while
	// sessions.enforce is still false: the "missing" and "invalid" series are
	// exactly the clicks that flipping it would start refusing.
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
		planetv1connect.ClickServiceClickProcedure,
		// A bonus is only ever spent as clicks, which need a session — so the
		// box that grants them asks for one too, rather than being the one way
		// to widen an allowance without proving anything.
		planetv1connect.ClickServiceClaimBonusProcedure,
		// A bomb writes the map, so it asks for what a click does.
		planetv1connect.ClickServiceDropBombProcedure,
		// A refill fills the allowance clicks spend, so it asks for what a click does.
		planetv1connect.ClickServiceUseRefillProcedure,
		// A right answer grants the same charge a caught box does, so the question asks for the
		// same thing the box does. Opening is gated too, and not only for symmetry: opening is what
		// starts the clock, and the caller it starts it for has to be the one that answers.
		planetv1connect.ClickServiceOpenQuizProcedure,
		planetv1connect.ClickServiceAnswerQuizProcedure,
	)
}

// NewSessionReaderInterceptor reads a token when the client sends one, and refuses nothing. On GetBudget the
// budget is then the account's, and on GetCharges the charges; on ListenForEvents the stream knows whose
// account it serves. Without a token, or
// with a bad one, a client that has not minted yet reads its scope's allowance and follows the planet as before.
func NewSessionReaderInterceptor(verifier ClickSessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		planetv1connect.ClickServiceGetBudgetProcedure,
		planetv1connect.ClickServiceGetChargesProcedure,
		planetv1connect.ClickServiceListenForEventsProcedure,
	)
}
