package get_mute_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var ErrInvalidAccount = errors.New("the account id is not a UUID")

type Query interface {
	Mute(ctx context.Context, account messages.AccountID) (*chatv1.GetMuteResponse, error)
}

func New(query Query) GetMuteHandler {
	return GetMuteHandler{query: query}
}

type GetMuteHandler struct {
	query Query
}

func (h GetMuteHandler) GetMute(
	ctx context.Context,
	req *connect.Request[chatv1.GetMuteRequest],
) (*connect.Response[chatv1.GetMuteResponse], error) {
	account := messages.AccountIDOf(req.Msg.GetAccountId())
	if account == messages.NoAccount {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidAccount)
	}

	mute, err := h.query.Mute(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the mute: %w", err)
	}

	res := connect.NewResponse(mute)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
