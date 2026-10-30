package keep_video_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/keep_video_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in keep_video_usecase.In) error
}

func New(useCase UseCase) KeepVideoHandler {
	return KeepVideoHandler{useCase: useCase}
}

type KeepVideoHandler struct {
	useCase UseCase
}

func (h KeepVideoHandler) KeepVideo(
	ctx context.Context,
	req *connect.Request[clipsv1.KeepVideoRequest],
) (*connect.Response[clipsv1.KeepVideoResponse], error) {
	id, err := videos.VideoIDOf(req.Msg.GetVideoId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = h.useCase.Execute(ctx, keep_video_usecase.In{ID: id, Name: req.Msg.GetName(), Caption: req.Msg.GetCaption()})
	switch {
	case errors.Is(err, videos.ErrInvalidVideo):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, videos.ErrNotUploaded):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, videos.ErrVideosOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, videos.ErrVideosOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&clipsv1.KeepVideoResponse{}), nil
}
