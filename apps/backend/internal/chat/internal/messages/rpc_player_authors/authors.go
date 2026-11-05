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
	GetPlayers(
		ctx context.Context,
		req *connect.Request[playerv1.GetPlayersRequest],
	) (*connect.Response[playerv1.GetPlayersResponse], error)
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
	return messages.AuthorOf(player.GetName(), player.GetAdmin(), int32(player.GetColor())), nil
}

func (a *Authors) Authors(
	ctx context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	found := make(map[messages.AccountID]messages.Author, len(accounts))
	if len(accounts) == 0 {
		return found, nil
	}

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.GetPlayers(ctx,
		connect.NewRequest(&playerv1.GetPlayersRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, player := range res.Msg.GetPlayers() {
		found[messages.AccountIDOf(player.GetAccountId())] = messages.AuthorOf(player.GetName(), player.GetAdmin(), int32(player.GetColor()))
	}
	return found, nil
}
