// Package rpc_player_authors asks the player module who posts, over the internal listener.
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

// Dialer is cpbootstrap's internal listener: the only way one module reaches another.
type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

// askTimeout bounds one call, which a post waits on. It is loopback, so this is a stuck socket rather than a
// slow network, and the post is refused.
const askTimeout = time.Second

type Authors struct {
	dial Dialer
}

func New(dial Dialer) *Authors {
	return &Authors{dial: dial}
}

// Author asks on every post, and keeps nothing: a player may choose a name at any time, and posts are few.
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

	return messages.Author{Name: res.Msg.GetName(), Admin: res.Msg.GetAdmin()}, nil
}

// Authors is who each of these accounts is, asked once for a whole page of them. An account the player module
// cannot name is absent from the answer: it was deleted, and messages.Named decides what stands in its place.
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
			Name:  author.GetName(),
			Admin: author.GetAdmin(),
		}
	}
	return found, nil
}
