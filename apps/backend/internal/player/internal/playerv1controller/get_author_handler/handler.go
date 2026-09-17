// Package get_author_handler serves player.v1.InternalService/GetAuthor, for the other modules.
package get_author_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type UseCase interface {
	Execute(ctx context.Context, in get_author_usecase.In) (players.Author, error)
}

func New(useCase UseCase) GetAuthorHandler {
	return GetAuthorHandler{useCase: useCase}
}

type GetAuthorHandler struct {
	useCase UseCase
}

// GetAuthor reads an id that is not an account as no account.
func (h GetAuthorHandler) GetAuthor(
	ctx context.Context,
	req *connect.Request[playerv1.GetAuthorRequest],
) (*connect.Response[playerv1.GetAuthorResponse], error) {
	account, err := players.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		account = cpsession.NoAccount
	}

	author, err := h.useCase.Execute(ctx, get_author_usecase.In{Account: account, IP: req.Msg.GetIp()})
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(&playerv1.GetAuthorResponse{
		Username: string(author.Name),
		Tag:      string(author.Tag),
		Admin:    author.Admin,
	}), nil
}
