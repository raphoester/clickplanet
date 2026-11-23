package get_email_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Query interface {
	Email(ctx context.Context, account players.AccountID) (*playerv1.GetEmailResponse, error)
}

func New(query Query) GetEmailHandler {
	return GetEmailHandler{query: query}
}

type GetEmailHandler struct {
	query Query
}

func (h GetEmailHandler) GetEmail(
	ctx context.Context,
	req *connect.Request[playerv1.GetEmailRequest],
) (*connect.Response[playerv1.GetEmailResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	email, err := h.query.Email(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(email), nil
}
