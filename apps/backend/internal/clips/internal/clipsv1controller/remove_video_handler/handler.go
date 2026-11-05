package remove_video_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

type UseCase interface {
	Execute(ctx context.Context, id videos.VideoID) error
}

func New(useCase UseCase) RemoveVideoHandler {
	return RemoveVideoHandler{useCase: useCase}
}

type RemoveVideoHandler struct {
	useCase UseCase
}

func (h RemoveVideoHandler) RemoveVideo(
	ctx context.Context,
	req *connect.Request[clipsv1.RemoveVideoRequest],
) (*connect.Response[clipsv1.RemoveVideoResponse], error) {
	id, err := videos.VideoIDOf(req.Msg.GetVideoId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = h.useCase.Execute(ctx, id)
	switch {
	case errors.Is(err, videos.ErrNoVideo):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, videos.ErrVideosOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, videos.ErrVideosOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&clipsv1.RemoveVideoResponse{}), nil
}
