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
	GetAuthor(ctx context.Context, req *connect.Request[playerv1.GetAuthorRequest]) (*connect.Response[playerv1.GetAuthorResponse], error)
	GetAuthors(
		ctx context.Context,
		req *connect.Request[playerv1.GetAuthorsRequest],
	) (*connect.Response[playerv1.GetAuthorsResponse], error)
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

	res, err := a.player.GetAuthor(ctx,
		connect.NewRequest(&playerv1.GetAuthorRequest{AccountId: account.String()}))
	if err != nil {
		return messages.Author{}, fmt.Errorf("failed to ask the player module who posts: %w", err)
	}

	return messages.AuthorOf(res.Msg.GetName(), res.Msg.GetAdmin(), int32(res.Msg.GetColor()), res.Msg.GetStreak(), titleOf(res.Msg.GetWornTitle())), nil
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

	res, err := a.player.GetAuthors(ctx,
		connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, author := range res.Msg.GetAuthors() {
		found[messages.AccountIDOf(author.GetAccountId())] = messages.AuthorOf(author.GetName(), author.GetAdmin(), int32(author.GetColor()), author.GetStreak(), titleOf(author.GetWornTitle()))
	}
	return found, nil
}

func titleOf(title *playerv1.Title) messages.Title {
	rank := title.GetRank()
	return messages.TitleOf(title.GetId(), title.GetName(),
		messages.RankOf(rank.GetTrackId(), rank.GetTrackName(), rank.GetNumber(), rank.GetCount()))
}
