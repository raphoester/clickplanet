package create_video_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos/usecases/create_upload_usecase"
)

type UseCase interface {
	Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error)
}

func New(useCase UseCase) CreateVideoUploadHandler {
	return CreateVideoUploadHandler{useCase: useCase}
}

type CreateVideoUploadHandler struct {
	useCase UseCase
}

func (h CreateVideoUploadHandler) CreateVideoUpload(
	_ context.Context,
	req *connect.Request[clipsv1.CreateVideoUploadRequest],
) (*connect.Response[clipsv1.CreateVideoUploadResponse], error) {
	out, err := h.useCase.Execute(create_upload_usecase.In{
		MediaType: videos.MediaType(req.Msg.GetContentType()),
		Size:      int64(min(req.Msg.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
	})
	switch {
	case errors.Is(err, videos.ErrInvalidVideo):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, videos.ErrVideosOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, videos.ErrVideosOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&clipsv1.CreateVideoUploadResponse{VideoId: out.ID.String(), UploadUrl: out.URL}), nil
}
