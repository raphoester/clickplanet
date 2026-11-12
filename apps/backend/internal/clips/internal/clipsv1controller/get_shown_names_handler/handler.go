package get_shown_names_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

// MaxAsked is how many accounts one call may name: a video's makers, never the whole game.
const MaxAsked = 100

var ErrTooMany = errors.New("too many accounts asked at once")

type Query interface {
	ShownNames(ctx context.Context, accounts []casts.AccountID) (*clipsv1.GetShownNamesResponse, error)
}

func New(query Query) GetShownNamesHandler {
	return GetShownNamesHandler{query: query}
}

type GetShownNamesHandler struct {
	query Query
}

func (h GetShownNamesHandler) GetShownNames(
	ctx context.Context,
	req *connect.Request[clipsv1.GetShownNamesRequest],
) (*connect.Response[clipsv1.GetShownNamesResponse], error) {
	ids := req.Msg.GetAccountIds()
	if len(ids) > MaxAsked {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d, at most %d", ErrTooMany, len(ids), MaxAsked))
	}
	asked := make([]casts.AccountID, 0, len(ids))
	for _, id := range ids {
		account, err := casts.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	answer, err := h.query.ShownNames(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(answer)
	// A rename, an opt-out or the operator's verdict holds at the next read.
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
