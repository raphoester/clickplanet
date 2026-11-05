package get_teaser_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
)

type Query interface {
	Teaser(ctx context.Context, battleID string) (*promov1.GetTeaserResponse, error)
}

func New(query Query) GetTeaserHandler {
	return GetTeaserHandler{query: query}
}

type GetTeaserHandler struct {
	query Query
}

func (h GetTeaserHandler) GetTeaser(
	ctx context.Context,
	req *connect.Request[promov1.GetTeaserRequest],
) (*connect.Response[promov1.GetTeaserResponse], error) {
	teaser, err := h.query.Teaser(ctx, req.Msg.GetBattleId())
	switch {
	case errors.Is(err, teaser_query.ErrNoBattle):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(teaser), nil
}
