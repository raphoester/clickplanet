package websocket_publisher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPublisherServesConnectedClients(t *testing.T) {
	updates, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)
	defer func() { _ = conn.CloseNow() }()

	updates <- domain.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}

	typ, bin, err := conn.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, websocket.MessageBinary, typ)

	var update planetv1.TileUpdate
	require.NoError(t, proto.Unmarshal(bin, &update))
	assert.Equal(t, uint32(42), update.TileId)
	assert.Equal(t, "fr", update.CountryId)
	assert.Equal(t, "de", update.PreviousCountryId)
}

func TestPublisherForgetsAbruptlyDisconnectedClients(t *testing.T) {
	updates, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)

	require.NoError(t, conn.CloseNow()) // killed without a close handshake

	requireEventualClientCount(t, publisher, 0)

	// the fanout still works once the dead client is gone
	updates <- domain.TileUpdate{Tile: 1, Value: "fr"}
	requireClientCount(t, publisher, 0)
}

func TestPublisherForgetsClientsThatCloseCleanly(t *testing.T) {
	_, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)

	require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))

	requireEventualClientCount(t, publisher, 0)
}

func startPublisher(t *testing.T) (chan<- domain.TileUpdate, *Publisher, string) {
	t.Helper()

	updates := make(chan domain.TileUpdate)
	publisher := New(updates, logging.NewSLogger())
	go publisher.Run()

	router := http.NewServeMux()
	publisher.DeclareRoutes(router)
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
		close(updates)
	})

	return updates, publisher, "ws" + strings.TrimPrefix(server.URL, "http") + "/listen"
}

func clientCount(p *Publisher) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

func requireClientCount(t *testing.T, p *Publisher, expected int) {
	t.Helper()
	require.Equal(t, expected, clientCount(p))
}

func requireEventualClientCount(t *testing.T, p *Publisher, expected int) {
	t.Helper()
	require.Eventually(t, func() bool {
		return clientCount(p) == expected
	}, 5*time.Second, 10*time.Millisecond)
}

// dialClient connects to the publisher and waits for the connection to be
// registered: websocket.Accept writes the 101 response before Subscribe adds
// the client, so the dial returning does not mean the client is known yet.
func dialClient(t *testing.T, ctx context.Context, p *Publisher, url string) *websocket.Conn {
	t.Helper()

	conn, _, err := websocket.Dial(ctx, url, nil)
	require.NoError(t, err)

	requireEventualClientCount(t, p, 1)
	return conn
}
