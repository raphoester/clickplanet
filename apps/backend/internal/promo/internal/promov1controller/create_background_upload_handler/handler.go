package create_background_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/create_upload_usecase"
)

type UseCase interface {
	Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error)
}

func New(useCase UseCase) CreateBackgroundUploadHandler {
	return CreateBackgroundUploadHandler{useCase: useCase}
}

type CreateBackgroundUploadHandler struct {
	useCase UseCase
}

func (h CreateBackgroundUploadHandler) CreateBackgroundUpload(
	_ context.Context,
	req *connect.Request[promov1.CreateBackgroundUploadRequest],
) (*connect.Response[promov1.CreateBackgroundUploadResponse], error) {
	out, err := h.useCase.Execute(create_upload_usecase.In{
		MediaType: backgrounds.MediaType(req.Msg.GetContentType()),
		Size:      int64(min(req.Msg.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
	})
	switch {
	case errors.Is(err, backgrounds.ErrInvalidBackground):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, backgrounds.ErrBackgroundsOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, backgrounds.ErrBackgroundsOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&promov1.CreateBackgroundUploadResponse{BackgroundId: out.ID.String(), UploadUrl: out.URL}), nil
}
