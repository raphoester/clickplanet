package get_post_links_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
)

// At most a week of posts a player is named in: far more than any read asks.
const maxPosts = 500

var errTooMany = errors.New("more than 500 posts asked at once")

type Query interface {
	Links(ctx context.Context, posts []string) (*publishingv1.GetPostLinksResponse, error)
}

func New(query Query) GetPostLinksHandler {
	return GetPostLinksHandler{query: query}
}

type GetPostLinksHandler struct {
	query Query
}

func (h GetPostLinksHandler) GetPostLinks(
	ctx context.Context,
	req *connect.Request[publishingv1.GetPostLinksRequest],
) (*connect.Response[publishingv1.GetPostLinksResponse], error) {
	if len(req.Msg.GetPostIds()) > maxPosts {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooMany)
	}
	answer, err := h.query.Links(ctx, req.Msg.GetPostIds())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(answer), nil
}
