package planetv1controller

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type fakeVerifier struct {
	valid map[string]cpsession.ID
	asked []string
}

func (v *fakeVerifier) Verify(token string, _ string, _ time.Time) (cpsession.ID, error) {
	v.asked = append(v.asked, token)

	id, ok := v.valid[token]
	if !ok {
		return "", errors.New("no")
	}
	return id, nil
}

type sessionResult struct {
	ran       bool
	sessionID string
	registry  prometheus.Gatherer
	err       error
}

func checkSession(
	t *testing.T,
	verifier ClickSessionVerifier,
	enforce bool,
	procedure string,
	token string,
) sessionResult {
	t.Helper()

	registry := prometheus.NewRegistry()
	interceptor, err := NewSessionInterceptor(verifier, nil, enforce, registry)
	require.NoError(t, err)

	result := sessionResult{registry: registry}
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		result.ran = true
		result.sessionID = cpctx.GetSessionID(ctx)
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	header := http.Header{}
	if token != "" {
		header.Set(cpconnect.SessionHeader, token)
	}

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}, header: header}

	_, result.err = interceptor.WrapUnary(next)(t.Context(), req)

	return result
}

func validVerifier() *fakeVerifier {
	return &fakeVerifier{valid: map[string]cpsession.ID{"good-token": "abcd1234"}}
}

func TestSessionInterceptorWhenEnforcing(t *testing.T) {
	const enforce = true

	t.Run("lets a click with a valid session through and puts its id on the context", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "good-token")

		require.NoError(t, result.err)
		require.True(t, result.ran)
		require.Equal(t, "abcd1234", result.sessionID)
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "valid"), 1e-9)
	})

	t.Run("refuses a click carrying no session at all", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "")

		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(result.err))
		require.ErrorIs(t, result.err, ErrNoSession)
		require.False(t, result.ran, "a refused click must not reach the domain")
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "missing"), 1e-9)
	})

	t.Run("refuses a click carrying a session it did not mint", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "forged")

		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(result.err))
		require.False(t, result.ran)
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "invalid"), 1e-9)
	})

	t.Run("leaves the read procedures alone", func(t *testing.T) {
		verifier := validVerifier()

		for _, procedure := range []string{
			planetv1connect.ClickServiceGetMapProcedure,
			planetv1connect.ClickServiceMapDensityProcedure,
		} {
			result := checkSession(t, verifier, enforce, procedure, "")

			require.NoError(t, result.err, "a session is never needed to load or watch the planet")
			require.True(t, result.ran)
		}

		require.Empty(t, verifier.asked, "reads are never even offered to the verifier")
	})
}

// The mode this ships in: it decides nothing, so a client that predates
// sessions keeps working, and the counter says what turning it on would cost.
func TestSessionInterceptorWhenObserving(t *testing.T) {
	const enforce = false

	t.Run("lets a click carrying no session through, and counts it", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "")

		require.NoError(t, result.err)
		require.True(t, result.ran)
		require.Empty(t, result.sessionID)
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "missing"), 1e-9)
	})

	t.Run("lets a click carrying a forged session through, and counts it", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "forged")

		require.NoError(t, result.err)
		require.True(t, result.ran)
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "invalid"), 1e-9)
	})

	t.Run("still reports the id of a session it did mint", func(t *testing.T) {
		result := checkSession(t, validVerifier(), enforce, planetv1connect.ClickServiceClickProcedure, "good-token")

		require.NoError(t, result.err)
		require.Equal(t, "abcd1234", result.sessionID)
		require.InDelta(t, 1.0, sessionChecks(t, result.registry, "valid"), 1e-9)
	})
}

// Same reason the blocklist sits outside the throttle: a click refused for its
// session must not also spend a token, or the retry that follows the mint would
// come back 429 and the player would be shown the throttle dialog instead.
func TestSessionCheckRunsBeforeTheThrottle(t *testing.T) {
	interceptor, err := NewSessionInterceptor(validVerifier(), nil, true, prometheus.NewRegistry())
	require.NoError(t, err)

	limiter := &fakeLimiter{allow: true}
	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		interceptor,
		NewRateLimitInterceptor(limiter),
	))

	require.Equal(t, http.StatusUnauthorized, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, limiter.keys, "the refused click never reached the bucket")
}

func TestSessionRefusalIsA401OverHTTP(t *testing.T) {
	interceptor, err := NewSessionInterceptor(validVerifier(), nil, true, prometheus.NewRegistry())
	require.NoError(t, err)

	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		interceptor,
		NewRateLimitInterceptor(allowAll{}),
	))

	click := func(token string) error {
		req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
		req.Header().Set("X-Real-IP", "1.2.3.4")
		if token != "" {
			req.Header().Set(cpconnect.SessionHeader, token)
		}
		_, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).
			Click(t.Context(), req)
		return err
	}

	require.NoError(t, click("good-token"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(click("")))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(click("forged")))
	require.Equal(t, http.StatusUnauthorized, clickStatus(t, server, "1.2.3.4"))
}

func sessionChecks(t *testing.T, gatherer prometheus.Gatherer, verdict string) float64 {
	t.Helper()

	families, err := gatherer.Gather()
	require.NoError(t, err)

	found := prometheus.NewCounter(prometheus.CounterOpts{Name: "found"})
	for _, family := range families {
		if family.GetName() != "click_session_checks" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "verdict" && label.GetValue() == verdict {
					found.Add(metric.GetCounter().GetValue())
				}
			}
		}
	}

	return testutil.ToFloat64(found)
}
