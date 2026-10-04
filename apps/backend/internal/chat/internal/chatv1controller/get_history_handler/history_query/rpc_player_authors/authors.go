package rpc_player_authors

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = time.Second

func New(dial Dialer) *Authors {
	return &Authors{dial: dial}
}

type Authors struct {
	dial Dialer
}

var _ history_query.Authors = (*Authors)(nil)

func (a *Authors) Authors(
	ctx context.Context,
	accounts []cpsession.AccountID,
) (map[cpsession.AccountID]*playerv1.Author, error) {
	found := make(map[cpsession.AccountID]*playerv1.Author, len(accounts))
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
		id, err := uuid.Parse(author.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("the player module answered for %q, which is not an account: %w", author.GetAccountId(), err)
		}
		found[cpsession.AccountID(id)] = author
	}
	return found, nil
}
