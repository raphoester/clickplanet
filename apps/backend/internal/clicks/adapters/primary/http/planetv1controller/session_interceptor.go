package planetv1controller

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// ErrNoSession is a refusal the player never reads: the client mints a token
// and retries the click. It is only visible to something that is not the
// client, which is the point.
var ErrNoSession = errors.New("clicks require a session; call session.v1.SessionService/CreateSession first")

type ClickSessionVerifier = connectutil.SessionVerifier

func NewSessionInterceptor(
	verifier ClickSessionVerifier,
	timeProvider xtime.Provider,
	enforce bool,
	registerer prometheus.Registerer,
) (connect.Interceptor, error) {
	// Labelled per verdict so the cost of enforcing is visible while
	// sessions.enforce is still false: the "missing" and "invalid" series are
	// exactly the clicks that flipping it would start refusing.
	checks := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "click_session_checks",
		Help: "Clicks by the verdict on the session token they carried",
	}, []string{"verdict"})

	if err := registerer.Register(checks); err != nil {
		return nil, fmt.Errorf("failed to register counter: %w", err)
	}

	return connectutil.NewSessionInterceptor(
		verifier,
		timeProvider,
		ErrNoSession,
		enforce,
		func(verdict connectutil.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		planetv1connect.ClickServiceClickProcedure,
		// A bonus is only ever spent as clicks, which need a session — so the
		// box that grants them asks for one too, rather than being the one way
		// to widen an allowance without proving anything.
		planetv1connect.ClickServiceClaimBonusProcedure,
	), nil
}
