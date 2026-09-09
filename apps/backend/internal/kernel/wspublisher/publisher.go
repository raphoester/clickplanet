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

const writeTimeout = 2 * time.Second

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

	closed := conn.CloseRead(context.Background())
	go func() {
		<-closed.Done()
		p.removeClient(conn)
	}()
}

func (p *Publisher[T]) removeClient(conn *websocket.Conn) {
	p.mu.Lock()
	_, registered := p.clients[conn]
	delete(p.clients, conn)
	p.mu.Unlock()

	if !registered {
		return
	}

	_ = conn.CloseNow()
}

func (p *Publisher[T]) DeclareRoutes(router *http.ServeMux) {
	router.HandleFunc(p.route, p.Subscribe)
}
