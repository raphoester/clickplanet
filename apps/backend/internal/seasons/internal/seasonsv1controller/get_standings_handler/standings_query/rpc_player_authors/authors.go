package rpc_player_authors

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler/standings_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Player interface {
	GetAuthors(
		ctx context.Context,
		req *connect.Request[playerv1.GetAuthorsRequest],
	) (*connect.Response[playerv1.GetAuthorsResponse], error)
}

const askTimeout = time.Second

func New(player Player) *Authors {
	return &Authors{player: player}
}

type Authors struct {
	player Player
}

var _ standings_query.Authors = (*Authors)(nil)

func (a *Authors) Authors(
	ctx context.Context,
	accounts []standings.AccountID,
) (map[standings.AccountID]*playerv1.Author, error) {
	found := make(map[standings.AccountID]*playerv1.Author, len(accounts))
	if len(accounts) == 0 {
		return found, nil
	}

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.player.GetAuthors(ctx, connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the player module who these accounts are: %w", err)
	}

	for _, author := range res.Msg.GetAuthors() {
		id, err := uuid.Parse(author.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("the player module answered for %q, which is not an account: %w", author.GetAccountId(), err)
		}
		found[standings.AccountID(id)] = author
	}
	return found, nil
}
