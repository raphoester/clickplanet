// Package get_author_handler serves player.v1.InternalService/GetAuthor, for the other modules.
package get_author_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (players.Author, error)
}

func New(useCase UseCase) GetAuthorHandler {
	return GetAuthorHandler{useCase: useCase}
}

type GetAuthorHandler struct {
	useCase UseCase
}

// GetAuthor refuses an id that is not an account: only an account has a name.
func (h GetAuthorHandler) GetAuthor(
	ctx context.Context,
	req *connect.Request[playerv1.GetAuthorRequest],
) (*connect.Response[playerv1.GetAuthorResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	author, err := h.useCase.Execute(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&playerv1.GetAuthorResponse{
		Name:  author.Name,
		Admin: author.Admin,
	}), nil
}
