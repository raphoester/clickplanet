package create_submission_upload_handler

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/create_submission_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(in create_submission_upload_usecase.In) (create_upload_usecase.Out, error)
}

func New(useCase UseCase) CreateSubmissionUploadHandler {
	return CreateSubmissionUploadHandler{useCase: useCase}
}

type CreateSubmissionUploadHandler struct {
	useCase UseCase
}

func (h CreateSubmissionUploadHandler) CreateSubmissionUpload(
	ctx context.Context,
	req *connect.Request[flagsv1.CreateSubmissionUploadRequest],
) (*connect.Response[flagsv1.CreateSubmissionUploadResponse], error) {
	out, err := h.useCase.Execute(create_submission_upload_usecase.In{
		Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx), ContentType: req.Msg.GetContentType(),
		Size: int64(min(req.Msg.GetSizeBytes(), math.MaxInt64)), //nolint:gosec // capped on this line.
	})
	switch {
	case errors.Is(err, submissions.ErrGuest):
		return nil, connect.NewError(connect.CodePermissionDenied, submissions.ErrGuest)
	case errors.Is(err, pictures.ErrNotAPicture), errors.Is(err, pictures.ErrTooLarge):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pictures.ErrOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, pictures.ErrOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.CreateSubmissionUploadResponse{
		UploadId: out.Upload.String(), UploadUrl: out.URL,
	}), nil
}
