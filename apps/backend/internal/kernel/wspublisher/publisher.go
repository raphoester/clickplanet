// Package wspublisher fans a stream of values out to every connected WebSocket
// client. It knows nothing about what it is carrying: the payload type and its
// encoding are the caller's, so a bounded context can broadcast its own
// messages without either context depending on the other.
package wspublisher

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

// writeTimeout bounds how long a single client can hold up the fanout. A client
// that cannot absorb an update within that delay is dropped.
const writeTimeout = 2 * time.Second

// New builds a publish-only endpoint serving route. Each stream gets its own
// route rather than sharing one socket: frames carry a bare protobuf message
// with no type tag, so a second kind of payload on an existing route would be
// indistinguishable from the first to every already-deployed client.
func New[T any](
	updates <-chan T,
	route string,
	encode func(T) ([]byte, error),
	logger logging.Logger,
) *Publisher[T] {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	return &Publisher[T]{
		clients: make(map[*websocket.Conn]*clientMD),
		updates: updates,
		route:   route,
		encode:  encode,
		logger:  logger,
	}
}

type Publisher[T any] struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]*clientMD
	updates <-chan T
	route   string
	encode  func(T) ([]byte, error)
	logger  logging.Logger
}

type clientMD struct {
}

func (p *Publisher[T]) Run() {
	for update := range p.updates {
		bin, err := p.encode(update)
		if err != nil {
			p.logger.Error("failed to encode an update for the websocket fanout",
				lf.String("route", p.route),
				lf.Err(err),
			)
			continue
		}

		var dead []*websocket.Conn

		p.mu.RLock()
		for client := range p.clients {
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := client.Write(ctx, websocket.MessageBinary, bin)
			cancel()
			if err != nil {
				dead = append(dead, client)
			}
		}
		p.mu.RUnlock()

		// Dropped outside the loop: removal needs the write lock, which cannot
		// be taken while the read lock above is held.
		for _, client := range dead {
			p.removeClient(client)
		}
	}
}

func (p *Publisher[T]) Subscribe(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:     []string{"*"},
		InsecureSkipVerify: true,
	})

	if err != nil {
		p.logger.Error("failed to accept websocket connection",
			lf.String("route", p.route),
			lf.Err(err),
		)
		http.Error(w, "cannot accept websocket connection", http.StatusInternalServerError)
		return
	}

	p.mu.Lock()
	p.clients[conn] = &clientMD{}
	p.mu.Unlock()

	// This connection is publish-only, but something has to read from it for
	// ping, pong and close frames to be handled. CloseRead does that in its own
	// goroutine and cancels the returned context as soon as the peer goes away,
	// so a client that disconnects between two updates is noticed right away
	// instead of on the next failed write.
	closed := conn.CloseRead(context.Background())
	go func() {
		<-closed.Done()
		p.removeClient(conn)
	}()
}

// removeClient drops a connection from the fanout and closes it. It is safe to
// call several times for the same connection, and from several goroutines.
func (p *Publisher[T]) removeClient(conn *websocket.Conn) {
	p.mu.Lock()
	_, registered := p.clients[conn]
	delete(p.clients, conn)
	p.mu.Unlock()

	if !registered { // someone else already closed it
		return
	}

	_ = conn.CloseNow()
}

func (p *Publisher[T]) DeclareRoutes(router *http.ServeMux) {
	router.HandleFunc(p.route, p.Subscribe)
}
