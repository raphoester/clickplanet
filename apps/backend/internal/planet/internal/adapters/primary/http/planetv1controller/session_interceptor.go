package planetv1controller

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"

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
	timeProvider cptime.Provider,
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

	return cpconnect.NewSessionInterceptor(
		verifier,
		timeProvider,
		ErrNoSession,
		enforce,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		planetv1connect.ClickServiceClickProcedure,
	), nil
}
