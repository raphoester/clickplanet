package playerv1controller

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

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
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
		return connect.NewResponse(&playerv1.GetStatsResponse{}), nil
	})

	inner := connect.NewRequest(&playerv1.GetStatsRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}
	interceptor := NewSessionInterceptor(verifier, cptime.SystemClock{}, prometheus.NewRegistry())
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, ran, err
}

func TestTheProceduresThatActForTheCallerNeedAClickToken(t *testing.T) {
	for _, procedure := range []string{
		playerv1connect.PlayerServiceSetNameProcedure,
		playerv1connect.PlayerServiceSetColorProcedure,
		playerv1connect.PlayerServiceAnnounceProcedure,
		playerv1connect.PlayerServiceWearTitleProcedure,
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
		playerv1connect.PlayerServiceGetProfileProcedure,
		playerv1connect.PlayerServiceGetStatsProcedure,
		playerv1connect.PlayerServiceLeaveProcedure,
		playerv1connect.PlayerServiceSetShownInClipsProcedure,
		playerv1connect.PlayerServiceGetTitlesProcedure,
		playerv1connect.PlayerServiceGetFrontsProcedure,
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

func TestAnyonesPlayerAndTheRosterNeedNoToken(t *testing.T) {
	for _, procedure := range []string{
		playerv1connect.PlayerServiceGetPlayerProcedure,
		playerv1connect.PlayerServiceGetRosterProcedure,
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

func (r accountRecorder) Execute(ctx context.Context, _ listen_for_events_usecase.Sink) error {
	r.accounts <- cpctx.GetAccount(ctx)
	return nil
}

type quietTitles struct{}

func (quietTitles) Execute(context.Context, players.AccountID, listen_for_titles_usecase.Sink) error {
	return nil
}

func TestAStreamOpenedWithAnIdentityTokenKnowsItsAccount(t *testing.T) {
	recorder := accountRecorder{accounts: make(chan string, 1)}

	mux := http.NewServeMux()
	mux.Handle(playerv1connect.NewPlayerServiceHandler(
		PlayerService{ListenForEventsHandler: listen_for_events_handler.New(recorder, quietTitles{})},
		connect.WithInterceptors(NewStreamSessionReader(verifier, cptime.SystemClock{})),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	listen := func(token string) string {
		req := connect.NewRequest(&playerv1.ListenForEventsRequest{})
		if token != "" {
			req.Header().Set(cpconnect.SessionHeader, token)
		}

		stream, err := playerv1connect.NewPlayerServiceClient(server.Client(), server.URL).ListenForEvents(t.Context(), req)
		require.NoError(t, err)
		for stream.Receive() {
		}
		require.NoError(t, stream.Err(), "a stream opened with no token or a bad one is refused nothing")
		require.NoError(t, stream.Close())

		return <-recorder.accounts
	}

	require.Equal(t, ada.String(), listen("an-identity-token"))
	require.Empty(t, listen(""))
	require.Empty(t, listen("forged"))
}
