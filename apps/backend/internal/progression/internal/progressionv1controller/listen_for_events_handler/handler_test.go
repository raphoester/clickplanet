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
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1/progressionv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/listen_for_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubTitles struct {
	earned chan titles.Standing
	asked  chan accounts.AccountID
}

func newStubTitles() stubTitles {
	return stubTitles{earned: make(chan titles.Standing, 1), asked: make(chan accounts.AccountID, 1)}
}

func (s stubTitles) Execute(ctx context.Context, account accounts.AccountID, sink listen_for_titles_usecase.Sink) error {
	s.asked <- account
	if err := sink.SendHeartbeat(); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}

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

func routes(earner stubTitles) http.Handler {
	handler := listen_for_events_handler.New(earner)
	mux := http.NewServeMux()
	mux.Handle(progressionv1connect.ProgressionServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		progressionv1connect.ProgressionServiceListenForEventsProcedure, handler.ListenForEvents,
	))
	return mux
}

func signedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(cpctx.AddAccountToContext(r.Context(), accounts.AccountID{15: 1}.String())))
	})
}

func listen(t *testing.T, handler http.Handler) *connect.ServerStreamForClient[progressionv1.ProgressionEvent] {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	stream, err := progressionv1connect.NewProgressionServiceClient(server.Client(), server.URL).ListenForEvents(ctx,
		connect.NewRequest(&progressionv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func TestAStreamWithNoAccountIsRefused(t *testing.T) {
	earner := newStubTitles()
	stream := listen(t, routes(earner))

	require.False(t, stream.Receive())
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(stream.Err()))
	assert.Empty(t, earner.asked, "a caller with no account follows no titles")
}

func TestAStreamWithAnAccountSaysItIsLiveThenSendsTheTitlesItEarns(t *testing.T) {
	earner := newStubTitles()
	stream := listen(t, signedIn(routes(earner)))

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat())
	conqueror, _ := titles.NewCatalog().StandingOf("conqueror")
	earner.earned <- conqueror

	require.True(t, stream.Receive(), stream.Err())
	assert.True(t, proto.Equal(&progressionv1.Title{
		Id: "conqueror", Name: "Conqueror", Rank: &progressionv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 4, Count: 5},
		Feat: "Took 100,000 tiles",
	}, stream.Msg().GetTitleEarned().GetTitle()), "%v", stream.Msg())
	assert.Equal(t, accounts.AccountID{15: 1}, <-earner.asked)
}
