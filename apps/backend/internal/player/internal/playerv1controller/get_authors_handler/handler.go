// Package get_authors_handler serves player.v1.InternalService/GetAuthors, for the other modules.
package get_authors_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type UseCase interface {
	Execute(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Author, error)
}

func New(useCase UseCase) GetAuthorsHandler {
	return GetAuthorsHandler{useCase: useCase}
}

type GetAuthorsHandler struct {
	useCase UseCase
}

// GetAuthors refuses an id that is not an account, as GetAuthor does: a caller holding a bad id has a bug, and
// answering the rest would hide it. An account nobody can name is left out of the answer instead.
func (h GetAuthorsHandler) GetAuthors(
	ctx context.Context,
	req *connect.Request[playerv1.GetAuthorsRequest],
) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	asked := make([]players.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		account, err := players.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	found, err := h.useCase.Execute(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	authors := make([]*playerv1.Author, 0, len(found))
	for account, author := range found {
		authors = append(authors, &playerv1.Author{
			AccountId: account.String(),
			Name:      author.Name,
			Admin:     author.Admin,
		})
	}

	res := connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: authors})
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
