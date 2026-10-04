package get_authors_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Query interface {
	Authors(ctx context.Context, accounts []players.AccountID) (*playerv1.GetAuthorsResponse, error)
}

func New(query Query) GetAuthorsHandler {
	return GetAuthorsHandler{query: query}
}

type GetAuthorsHandler struct {
	query Query
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

	authors, err := h.query.Authors(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(authors)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
