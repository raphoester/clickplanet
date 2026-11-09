package list_posts_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
)

// The service's own list, in the wire's words: the posting service's adapter answers it.
type Posts interface {
	Posts(ctx context.Context, since time.Time) ([]*publishingv1.Post, error)
}

func New(posts Posts) ListPostsHandler {
	return ListPostsHandler{posts: posts}
}

type ListPostsHandler struct {
	posts Posts
}

func (h ListPostsHandler) ListPosts(
	ctx context.Context,
	req *connect.Request[publishingv1.ListPostsRequest],
) (*connect.Response[publishingv1.ListPostsResponse], error) {
	if req.Msg.GetSince() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no since"))
	}
	listed, err := h.posts.Posts(ctx, req.Msg.GetSince().AsTime())
	var busy *posts.BusyError
	switch {
	case errors.As(err, &busy):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&publishingv1.ListPostsResponse{Posts: listed}), nil
}
