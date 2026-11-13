package rpc_planet_shares

import (
	"context"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Planet interface {
	GetTerritories(context.Context, *connect.Request[planetv1.GetTerritoriesRequest],
	) (*connect.Response[planetv1.GetTerritoriesResponse], error)
}

const askTimeout = 5 * time.Second

func New(planet Planet, every time.Duration) *Shares {
	shares := &Shares{planet: planet, every: every}
	shares.held.Store(&map[string]float64{})
	return shares
}

// Shares is the share of the map each flag holds, for the quiz's lean, as planet answered it last: asked every so
// often by a runner of its own, never while a question is drawn, which runs under the offers' lock. Until planet first
// answers, every share is 0, and the draw is flat.
type Shares struct {
	planet Planet
	every  time.Duration
	held   atomic.Pointer[map[string]float64]
}

func (s *Shares) Share(country string) float64 {
	return (*s.held.Load())[country]
}

func (s *Shares) Name() string { return "quiz-shares" }

func (s *Shares) Run(ctx context.Context) {
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()

	for {
		s.ask(ctx)
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// A failed ask keeps the shares planet answered last.
func (s *Shares) ask(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := s.planet.GetTerritories(ctx, connect.NewRequest(&planetv1.GetTerritoriesRequest{}))
	if err != nil || res.Msg.GetTiles() == 0 {
		return
	}

	held := make(map[string]float64, len(res.Msg.GetTerritories()))
	for _, territory := range res.Msg.GetTerritories() {
		held[territory.GetCountryId()] = float64(territory.GetTiles()) / float64(res.Msg.GetTiles())
	}
	s.held.Store(&held)
}
