package progressionv1controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1/progressionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ada = cpsession.AccountID{15: 1}

type tokenVerifier map[string]cpsession.Claims

func (v tokenVerifier) Verify(_ context.Context, token string, _ string, _ time.Time) (*cpsession.Claims, error) {
	claims, ok := v[token]
	if !ok {
		return nil, errors.New("not minted here")
	}
	return &claims, nil
}

var verifier = tokenVerifier{
	"a-click-token":     {ID: "a-mint", Account: ada, Attested: true},
	"an-identity-token": {ID: "a-resume", Account: ada},
}

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func run(t *testing.T, procedure string, token string) (string, bool, error) {
	t.Helper()

	var (
		account string
		ran     bool
	)
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account, ran = cpctx.GetAccount(ctx), true
		return connect.NewResponse(&progressionv1.GetStatsResponse{}), nil
	})

	inner := connect.NewRequest(&progressionv1.GetStatsRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}
	interceptor := NewSessionInterceptor(verifier, cptime.SystemClock{}, prometheus.NewRegistry())
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, ran, err
}

func TestTheProceduresThatActForTheCallerNeedAClickToken(t *testing.T) {
	for _, procedure := range []string{
		progressionv1connect.ProgressionServiceWearTitleProcedure,
	} {
		account, ran, err := run(t, procedure, "a-click-token")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
		require.Equal(t, ada.String(), account, procedure)

		_, ran, err = run(t, procedure, "an-identity-token")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.ErrorIs(t, err, ErrNoSession, procedure)
		require.False(t, ran, procedure)
	}
}

func TestTheReadsOfTheCallerTakeTheIdentityTokenOfAResumedSession(t *testing.T) {
	for _, procedure := range []string{
		progressionv1connect.ProgressionServiceGetStatsProcedure,
		progressionv1connect.ProgressionServiceGetTitlesProcedure,
		progressionv1connect.ProgressionServiceGetFrontsProcedure,
	} {
		account, ran, err := run(t, procedure, "an-identity-token")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
		require.Equal(t, ada.String(), account, procedure)

		_, ran, err = run(t, procedure, "")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.False(t, ran, procedure)
	}
}

func TestAnyonesPlayerTheRosterAndTheBadgesNeedNoToken(t *testing.T) {
	for _, procedure := range []string{
		progressionv1connect.ProgressionServiceGetProgressProcedure,
		progressionv1connect.ProgressionServiceGetBadgesProcedure,
	} {
		for _, token := range []string{"", "forged"} {
			_, ran, err := run(t, procedure, token)
			require.NoError(t, err, procedure)
			require.True(t, ran, procedure)
		}
	}
}

type accountRecorder struct {
	accounts chan string
}

func (r accountRecorder) ListenForEvents(
	ctx context.Context, _ *connect.Request[progressionv1.ListenForEventsRequest], _ *connect.ServerStream[progressionv1.ProgressionEvent],
) error {
	r.accounts <- cpctx.GetAccount(ctx)
	return nil
}

func TestAStreamOpenedWithAnIdentityTokenKnowsItsAccount(t *testing.T) {
	recorder := accountRecorder{accounts: make(chan string, 1)}

	mux := http.NewServeMux()
	mux.Handle(progressionv1connect.ProgressionServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		progressionv1connect.ProgressionServiceListenForEventsProcedure, recorder.ListenForEvents,
		connect.WithInterceptors(NewStreamSessionReader(verifier, cptime.SystemClock{})),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	listen := func(token string) string {
		req := connect.NewRequest(&progressionv1.ListenForEventsRequest{})
		if token != "" {
			req.Header().Set(cpconnect.SessionHeader, token)
		}

		stream, err := progressionv1connect.NewProgressionServiceClient(server.Client(), server.URL).ListenForEvents(t.Context(), req)
		require.NoError(t, err)
		for stream.Receive() {
		}
		require.NoError(t, stream.Err())
		require.NoError(t, stream.Close())

		return <-recorder.accounts
	}

	require.Equal(t, ada.String(), listen("an-identity-token"))
	require.Empty(t, listen(""), "the handler refuses it")
	require.Empty(t, listen("forged"))
}
