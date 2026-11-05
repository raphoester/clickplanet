package list_videos_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

type Query interface {
	Videos(ctx context.Context) (*clipsv1.ListVideosResponse, error)
}

func New(query Query) ListVideosHandler {
	return ListVideosHandler{query: query}
}

type ListVideosHandler struct {
	query Query
}

func (h ListVideosHandler) ListVideos(
	ctx context.Context,
	_ *connect.Request[clipsv1.ListVideosRequest],
) (*connect.Response[clipsv1.ListVideosResponse], error) {
	listed, err := h.query.Videos(ctx)
	switch {
	case errors.Is(err, videos.ErrVideosOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, videos.ErrVideosOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(listed), nil
}
