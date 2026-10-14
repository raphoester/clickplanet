package create_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/attachment"
)

type UseCase interface {
	Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error)
}

func New(useCase UseCase) CreateUploadHandler {
	return CreateUploadHandler{useCase: useCase}
}

type CreateUploadHandler struct {
	useCase UseCase
}

func (h CreateUploadHandler) CreateUpload(
	_ context.Context,
	req *connect.Request[supportv1.CreateUploadRequest],
) (*connect.Response[supportv1.CreateUploadResponse], error) {
	out, err := h.useCase.Execute(create_upload_usecase.In{
		MediaType: notes.MediaType(req.Msg.GetContentType()),
		Size:      int64(min(req.Msg.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
	})
	switch {
	case errors.Is(err, notes.ErrInvalidAttachment):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidAttachment)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	return connect.NewResponse(attachment.Uploaded(out.ID, out.URL)), nil
}
