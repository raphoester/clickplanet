package rpc_player_authors

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = time.Second

type Authors struct {
	dial Dialer
}

func New(dial Dialer) *Authors {
	return &Authors{dial: dial}
}

func (a *Authors) Author(ctx context.Context, account messages.AccountID) (messages.Author, error) {
	client, baseURL, err := a.dial.Dial()
	if err != nil {
		return messages.Author{}, fmt.Errorf("failed to reach the player module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := playerv1connect.NewInternalServiceClient(client, baseURL).GetAuthor(ctx,
		connect.NewRequest(&playerv1.GetAuthorRequest{AccountId: account.String()}))
	if err != nil {
		return messages.Author{}, fmt.Errorf("failed to ask the player module who posts: %w", err)
	}

	return messages.Author{
		Name:   res.Msg.GetName(),
		Admin:  res.Msg.GetAdmin(),
		Color:  int32(res.Msg.GetColor()),
		Streak: res.Msg.GetStreak(),
		Title:  titleOf(res.Msg.GetWornTitle()),
	}, nil
}

func (a *Authors) Authors(
	ctx context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	found := make(map[messages.AccountID]messages.Author, len(accounts))
	if len(accounts) == 0 {
		return found, nil
	}

	client, baseURL, err := a.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the player module: %w", err)
	}

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := playerv1connect.NewInternalServiceClient(client, baseURL).GetAuthors(ctx,
		connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, author := range res.Msg.GetAuthors() {
		found[messages.AccountIDOf(author.GetAccountId())] = messages.Author{
			Name:   author.GetName(),
			Admin:  author.GetAdmin(),
			Color:  int32(author.GetColor()),
			Streak: author.GetStreak(),
			Title:  titleOf(author.GetWornTitle()),
		}
	}
	return found, nil
}

func titleOf(title *playerv1.Title) messages.Title {
	rank := title.GetRank()
	return messages.Title{
		ID:   title.GetId(),
		Name: title.GetName(),
		Rank: messages.Rank{
			TrackID:   rank.GetTrackId(),
			TrackName: rank.GetTrackName(),
			Number:    rank.GetNumber(),
			Count:     rank.GetCount(),
		},
	}
}
