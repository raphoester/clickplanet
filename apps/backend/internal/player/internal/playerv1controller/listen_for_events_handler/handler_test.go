package listen_for_events_handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubRoster struct {
	end chan struct{}
}

func newStubRoster() stubRoster {
	return stubRoster{end: make(chan struct{})}
}

func (s stubRoster) Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error {
	if err := sink.SendRoster([]presence.Entry{ada}); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}

	select {
	case <-s.end:
	case <-ctx.Done():
	}
	return nil
}

type stubTitles struct {
	earned chan titles.Standing
	asked  chan players.AccountID
}

func newStubTitles() stubTitles {
	return stubTitles{earned: make(chan titles.Standing, 1), asked: make(chan players.AccountID, 1)}
}

func (s stubTitles) Execute(ctx context.Context, account players.AccountID, sink listen_for_titles_usecase.Sink) error {
	s.asked <- account

	for {
		select {
		case <-ctx.Done():
			return nil
		case standing := <-s.earned:
			if err := sink.SendTitleEarned(standing); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}

func routes(roster stubRoster, earner stubTitles) http.Handler {
	handler := listen_for_events_handler.New(roster, earner)
	mux := http.NewServeMux()
	mux.Handle(playerv1connect.PlayerServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		playerv1connect.PlayerServiceListenForEventsProcedure, handler.ListenForEvents))
	return mux
}

func signedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(cpctx.AddAccountToContext(r.Context(), players.AccountID{15: 1}.String())))
	})
}

func listen(t *testing.T, handler http.Handler) *connect.ServerStreamForClient[playerv1.PlayerEvent] {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	stream, err := playerv1connect.NewPlayerServiceClient(server.Client(), server.URL).ListenForEvents(ctx,
		connect.NewRequest(&playerv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func requireRoster(t *testing.T, stream *connect.ServerStreamForClient[playerv1.PlayerEvent]) {
	t.Helper()

	require.True(t, stream.Receive(), stream.Err())
	require.NotNil(t, stream.Msg().GetRoster(), "the roster comes first: %v", stream.Msg())
	require.Len(t, stream.Msg().GetRoster().GetEntries(), 1)
	assert.Equal(t, "k1", stream.Msg().GetRoster().GetEntries()[0].GetKey())
}

func TestAStreamWithNoAccountGetsTheRosterAndNoTitle(t *testing.T) {
	roster, earner := newStubRoster(), newStubTitles()
	conqueror, _ := titles.NewCatalog().StandingOf("conqueror")
	earner.earned <- conqueror
	stream := listen(t, routes(roster, earner))

	requireRoster(t, stream)
	close(roster.end)

	require.False(t, stream.Receive(), "no frame after the roster: %v", stream.Msg())
	require.NoError(t, stream.Err())
	assert.Empty(t, earner.asked, "a caller with no account follows no titles")
}

func TestAStreamWithAnAccountGetsTheRosterThenTheTitlesItEarns(t *testing.T) {
	roster, earner := newStubRoster(), newStubTitles()
	stream := listen(t, signedIn(routes(roster, earner)))

	requireRoster(t, stream)
	conqueror, _ := titles.NewCatalog().StandingOf("conqueror")
	earner.earned <- conqueror

	require.True(t, stream.Receive(), stream.Err())
	assert.Equal(t, "conqueror", stream.Msg().GetTitleEarned().GetTitle().GetId())
	assert.Equal(t, players.AccountID{15: 1}, <-earner.asked)
}
