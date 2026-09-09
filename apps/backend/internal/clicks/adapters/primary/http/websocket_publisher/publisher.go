package websocket_publisher

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"google.golang.org/protobuf/proto"
)

// writeTimeout bounds how long a single client can hold up the fanout.
// A client that cannot absorb an update within that delay is dropped, the same
// way the tile storage drops updates for subscribers that fall behind.
const writeTimeout = 2 * time.Second

func New(
	updates <-chan domain.TileUpdate,
	answerer *httpserver.Answerer,
) *Publisher {
	return &Publisher{
		clients:  make(map[*websocket.Conn]*clientMD),
		updates:  updates,
		answerer: answerer,
	}
}

type Publisher struct {
	mu       sync.RWMutex
	clients  map[*websocket.Conn]*clientMD
	updates  <-chan domain.TileUpdate
	answerer *httpserver.Answerer
}

type clientMD struct {
}

func (p *Publisher) Run() {
	for update := range p.updates {
		bin, err := proto.Marshal(&planetv1.TileUpdate{
			TileId:            update.Tile,
			CountryId:         update.Value,
			PreviousCountryId: update.Previous,
		})

		if err != nil {
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

		// Dropped outside of the loop: removal needs the write lock, which
		// cannot be taken while the read lock above is held.
		for _, client := range dead {
			p.removeClient(client)
		}
	}
}

func (p *Publisher) Subscribe(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:     []string{"*"},
		InsecureSkipVerify: true,
	})

	if err != nil {
		p.answerer.Err(w,
			fmt.Errorf("failed to accept websocket connection: %w", err),
			"cannot accept websocket connection",
			http.StatusInternalServerError,
		)
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
func (p *Publisher) removeClient(conn *websocket.Conn) {
	p.mu.Lock()
	_, registered := p.clients[conn]
	delete(p.clients, conn)
	p.mu.Unlock()

	if !registered { // someone else already closed it
		return
	}

	_ = conn.CloseNow()
}

func (p *Publisher) DeclareRoutes(router *http.ServeMux) {
	router.HandleFunc("/listen", p.Subscribe)
}
