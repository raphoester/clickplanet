package listen_for_events_handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/listen_for_events_handler"
)

func TestAPageHearsAFlagClosedWhileItsStreamIsOpen(t *testing.T) {
	board := feed.NewBoard(8)
	handler := listen_for_events_handler.New(listen_for_events_usecase.New(board, time.Hour), flagmessage.New("https://media.test"))
	mux := http.NewServeMux()
	mux.Handle(flagsv1connect.FlagServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		flagsv1connect.FlagServiceListenForEventsProcedure, handler.ListenForEvents))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	stream, err := flagsv1connect.NewFlagServiceClient(server.Client(), server.URL).
		ListenForEvents(t.Context(), connect.NewRequest(&flagsv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat(), "live at once")

	closedAt := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	board.Publish(feed.ToldAsItIs(catalog.Snapshot{
		ID: "r-0000000001", Name: "Rennes", Picture: catalog.TestPicture, State: catalog.Closed,
		DecidedAt: closedAt, EndsAt: closedAt,
	}))
	require.True(t, stream.Receive(), stream.Err())
	flag := stream.Msg().GetFlagChanged()
	assert.Equal(t, "r-0000000001", flag.GetCode())
	assert.True(t, flag.GetClosed())
	assert.Equal(t, "https://media.test/"+catalog.TestPicture.Key(), flag.GetSold().GetPicture().GetUrl())
	assert.Equal(t, closedAt, flag.GetSold().GetClosedAt().AsTime())

	board.Publish(feed.ToldGone("r-0000000001"))
	require.True(t, stream.Receive(), stream.Err())
	assert.Equal(t, "r-0000000001", stream.Msg().GetFlagDeleted().GetCode())
}
