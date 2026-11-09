package create_media_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
)

type UseCase interface {
	Execute(ctx context.Context, media posts.Media) (posts.MediaUpload, error)
}

func New(useCase UseCase) CreateMediaUploadHandler {
	return CreateMediaUploadHandler{useCase: useCase}
}

type CreateMediaUploadHandler struct {
	useCase UseCase
}

func (h CreateMediaUploadHandler) CreateMediaUpload(
	ctx context.Context,
	req *connect.Request[publishingv1.CreateMediaUploadRequest],
) (*connect.Response[publishingv1.CreateMediaUploadResponse], error) {
	upload, err := h.useCase.Execute(ctx, posts.Media{
		FileName: req.Msg.GetFileName(), ContentType: req.Msg.GetContentType(),
		Size: int64(min(req.Msg.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
	})
	if err != nil {
		return nil, errorOf(err)
	}
	return connect.NewResponse(&publishingv1.CreateMediaUploadResponse{
		UploadUrl: upload.UploadURL, MediaUrl: string(upload.Media),
	}), nil
}

func errorOf(err error) error {
	var busy *posts.BusyError
	switch {
	case errors.Is(err, posts.ErrInvalidMedia):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.As(err, &busy):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return err //nolint:wrapcheck // the error net answers it.
	}
}
