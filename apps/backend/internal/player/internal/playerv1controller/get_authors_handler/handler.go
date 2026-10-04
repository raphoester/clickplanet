package get_authors_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

type UseCase interface {
	Execute(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]wearing.Author, error)
}

func New(useCase UseCase) GetAuthorsHandler {
	return GetAuthorsHandler{useCase: useCase}
}

type GetAuthorsHandler struct {
	useCase UseCase
}

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
			Color:     playermessage.Color(author.Color),
			Streak:    author.Streak.Days,
			WornTitle: playermessage.Title(author.Worn),
		})
	}

	res := connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: authors})
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
