package create_flag_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/usecases/create_upload_usecase"
)

type UseCase interface {
	Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error)
}

func New(useCase UseCase) CreateFlagUploadHandler {
	return CreateFlagUploadHandler{useCase: useCase}
}

type CreateFlagUploadHandler struct {
	useCase UseCase
}

func (h CreateFlagUploadHandler) CreateFlagUpload(
	_ context.Context,
	req *connect.Request[flagsv1.CreateFlagUploadRequest],
) (*connect.Response[flagsv1.CreateFlagUploadResponse], error) {
	size := req.Msg.GetSizeBytes()
	if size > math.MaxInt64 {
		size = math.MaxInt64
	}
	out, err := h.useCase.Execute(create_upload_usecase.In{ContentType: req.Msg.GetContentType(), Size: int64(size)})
	switch {
	case errors.Is(err, pictures.ErrNotAPicture), errors.Is(err, pictures.ErrTooLarge):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pictures.ErrOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.CreateFlagUploadResponse{UploadId: out.Upload.String(), UploadUrl: out.URL}), nil
}
