package listen_for_events_handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/listen_for_events_handler"
)

func TestAPageHearsAFlagClosedWhileItsStreamIsOpen(t *testing.T) {
	board := feed.NewBoard(8)
	handler := listen_for_events_handler.New(listen_for_events_usecase.New(board, time.Hour))
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

	board.Publish(catalog.Snapshot{ID: "r-0000000001", Name: "Rennes", State: catalog.Closed, DecidedAt: time.Now()})
	require.True(t, stream.Receive(), stream.Err())
	assert.True(t, proto.Equal(&flagsv1.Flag{Code: "r-0000000001", Name: "Rennes", Closed: true},
		stream.Msg().GetFlagChanged()), "%v", stream.Msg())
}
