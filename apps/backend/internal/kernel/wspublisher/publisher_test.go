package wspublisher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payload stands in for whatever a bounded context broadcasts: the publisher
// only ever sees the bytes the encoder returns.
type payload struct {
	body string
	fail bool
}

func encodePayload(p payload) ([]byte, error) {
	if p.fail {
		return nil, errors.New("cannot encode")
	}
	return []byte(p.body), nil
}

func TestPublisherServesConnectedClients(t *testing.T) {
	updates, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)
	defer func() { _ = conn.CloseNow() }()

	updates <- payload{body: "hello"}

	typ, bin, err := conn.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, websocket.MessageBinary, typ)
	assert.Equal(t, "hello", string(bin))
}

// A payload the encoder refuses must not take the fanout down with it: the
// stream skips it and keeps serving whatever comes next.
func TestPublisherSkipsPayloadsItCannotEncode(t *testing.T) {
	updates, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)
	defer func() { _ = conn.CloseNow() }()

	updates <- payload{fail: true}
	updates <- payload{body: "next"}

	_, bin, err := conn.Read(ctx)
	require.NoError(t, err)
	assert.Equal(t, "next", string(bin))
}

func TestPublisherForgetsAbruptlyDisconnectedClients(t *testing.T) {
	updates, publisher, url := startPublisher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialClient(t, ctx, publisher, url)

	require.NoError(t, conn.CloseNow()) // killed without a close handshake

	requireEventualClientCount(t, publisher, 0)

	updates <- payload{body: "nobody is listening"}
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

func startPublisher(t *testing.T) (chan<- payload, *Publisher[payload], string) {
	t.Helper()

	updates := make(chan payload)
	publisher := New(updates, "/listen", encodePayload, logging.NewNopLogger())
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

func clientCount[T any](p *Publisher[T]) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

func requireClientCount[T any](t *testing.T, p *Publisher[T], expected int) {
	t.Helper()
	require.Equal(t, expected, clientCount(p))
}

func requireEventualClientCount[T any](t *testing.T, p *Publisher[T], expected int) {
	t.Helper()
	require.Eventually(t, func() bool {
		return clientCount(p) == expected
	}, 5*time.Second, 10*time.Millisecond)
}

func dialClient[T any](t *testing.T, ctx context.Context, p *Publisher[T], url string) *websocket.Conn {
	t.Helper()

	conn, _, err := websocket.Dial(ctx, url, nil)
	require.NoError(t, err)

	requireEventualClientCount(t, p, 1)
	return conn
}
