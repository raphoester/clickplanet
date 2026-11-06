package rpc_player_authors

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Player interface {
	IntroducePlayer(
		ctx context.Context,
		req *connect.Request[playerv1.IntroducePlayerRequest],
	) (*connect.Response[playerv1.IntroducePlayerResponse], error)
}

const askTimeout = time.Second

type Authors struct {
	player Player
}

func New(player Player) *Authors {
	return &Authors{player: player}
}

func (a *Authors) Author(ctx context.Context, account messages.AccountID) (messages.Author, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.IntroducePlayer(ctx,
		connect.NewRequest(&playerv1.IntroducePlayerRequest{AccountId: account.String()}))
	if err != nil {
		return messages.Author{}, fmt.Errorf("failed to ask the player module who posts: %w", err)
	}

	player := res.Msg.GetPlayer()
	if player.GetGuest() {
		return messages.GuestOf(player.GetName()), nil
	}
	return messages.AuthorOf(player.GetName()), nil
}
