package rpc_gamemaster_timeline

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

type Gamemaster interface {
	GetTimeline(context.Context, *connect.Request[gamemasterv1.GetTimelineRequest]) (*connect.Response[gamemasterv1.GetTimelineResponse], error)
}

func New(gamemaster Gamemaster) *Adapter {
	return &Adapter{gamemaster: gamemaster}
}

// The battles to come, in the gamemaster's words.
type Adapter struct {
	gamemaster Gamemaster
}

func (a *Adapter) Battles(ctx context.Context) ([]*gamemasterv1.Battle, error) {
	res, err := a.gamemaster.GetTimeline(ctx, connect.NewRequest(&gamemasterv1.GetTimelineRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the gamemaster the battles to come: %w", err)
	}
	return res.Msg.GetBattles(), nil
}
