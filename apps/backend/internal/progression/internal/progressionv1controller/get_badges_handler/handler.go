package get_badges_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

// MaxAsked is how many accounts one call may name: a page of names, never the whole game.
const MaxAsked = 100

var ErrTooMany = errors.New("too many accounts asked at once")

type Query interface {
	Badges(ctx context.Context, accounts []accounts.AccountID) (*progressionv1.GetBadgesResponse, error)
}

func New(query Query) GetBadgesHandler {
	return GetBadgesHandler{query: query}
}

type GetBadgesHandler struct {
	query Query
}

func (h GetBadgesHandler) GetBadges(
	ctx context.Context,
	req *connect.Request[progressionv1.GetBadgesRequest],
) (*connect.Response[progressionv1.GetBadgesResponse], error) {
	ids := req.Msg.GetAccountIds()
	if len(ids) > MaxAsked {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d, at most %d", ErrTooMany, len(ids), MaxAsked))
	}
	asked := make([]accounts.AccountID, 0, len(ids))
	for _, id := range ids {
		account, err := accounts.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	badges, err := h.query.Badges(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(badges)
	// The page keeps what it read: an answer kept on the way could undo what it then learns of its own.
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
