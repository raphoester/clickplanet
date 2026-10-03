package rpc_player_names

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = time.Second

type Players struct {
	dial Dialer
}

var _ standings.Players = (*Players)(nil)

func New(dial Dialer) *Players {
	return &Players{dial: dial}
}

func (p *Players) Players(ctx context.Context, accounts []standings.AccountID) (map[standings.AccountID]standings.Player, error) {
	found := make(map[standings.AccountID]standings.Player, len(accounts))
	if len(accounts) == 0 {
		return found, nil
	}

	client, baseURL, err := p.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the player module: %w", err)
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := playerv1connect.NewInternalServiceClient(client, baseURL).GetAuthors(ctx,
		connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, author := range res.Msg.GetAuthors() {
		account, err := standings.AccountIDOf(author.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("the player module answered for no account: %w", err)
		}
		found[account] = standings.Player{
			Name:  author.GetName(),
			Color: standings.Color(author.GetColor()),
			Guest: author.GetGuest(),
		}
	}
	return found, nil
}
