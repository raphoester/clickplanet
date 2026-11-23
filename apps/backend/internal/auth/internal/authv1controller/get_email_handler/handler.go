package get_email_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Query interface {
	Email(ctx context.Context, account accounts.AccountID) (*authv1.GetEmailResponse, error)
}

func New(query Query) GetEmailHandler {
	return GetEmailHandler{query: query}
}

type GetEmailHandler struct {
	query Query
}

func (h GetEmailHandler) GetEmail(
	ctx context.Context,
	req *connect.Request[authv1.GetEmailRequest],
) (*connect.Response[authv1.GetEmailResponse], error) {
	id, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	email, err := h.query.Email(ctx, id)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(email), nil
}
