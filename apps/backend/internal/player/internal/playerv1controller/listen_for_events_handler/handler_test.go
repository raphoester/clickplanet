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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
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

func routes(roster stubRoster) http.Handler {
	handler := listen_for_events_handler.New(roster)
	mux := http.NewServeMux()
	mux.Handle(playerv1connect.PlayerServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		playerv1connect.PlayerServiceListenForEventsProcedure, handler.ListenForEvents,
	))
	return mux
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

func TestAStreamGetsTheRosterFirst(t *testing.T) {
	roster := newStubRoster()
	stream := listen(t, routes(roster))

	requireRoster(t, stream)
	close(roster.end)

	require.False(t, stream.Receive(), "no frame after the roster: %v", stream.Msg())
	require.NoError(t, stream.Err())
}
