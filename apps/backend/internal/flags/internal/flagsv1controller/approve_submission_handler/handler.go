package approve_submission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in approve_submission_usecase.In) (submissions.Snapshot, error)
}

func New(useCase UseCase, messages submissionmessage.Messages) ApproveSubmissionHandler {
	return ApproveSubmissionHandler{useCase: useCase, messages: messages}
}

type ApproveSubmissionHandler struct {
	useCase  UseCase
	messages submissionmessage.Messages
}

func (h ApproveSubmissionHandler) ApproveSubmission(
	ctx context.Context,
	req *connect.Request[flagsv1.ApproveSubmissionRequest],
) (*connect.Response[flagsv1.ApproveSubmissionResponse], error) {
	approved, err := h.useCase.Execute(ctx, approve_submission_usecase.In{
		Submission: req.Msg.GetSubmissionId(), Moved: req.Msg.GetPin() != nil, Latitude: req.Msg.GetPin().GetLatitude(),
		Longitude: req.Msg.GetPin().GetLongitude(),
	})
	switch {
	case errors.Is(err, submissions.ErrUnknownSubmission):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, submissions.ErrDecided), errors.Is(err, catalog.ErrNameTaken):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, catalog.ErrInvalidPin):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pictures.ErrOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, pictures.ErrOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	kept, err := h.messages.Kept(approved)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.ApproveSubmissionResponse{Submission: kept}), nil
}
