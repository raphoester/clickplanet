package reject_submission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/reject_submission_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in reject_submission_usecase.In) (submissions.Snapshot, error)
}

func New(useCase UseCase, messages submissionmessage.Messages) RejectSubmissionHandler {
	return RejectSubmissionHandler{useCase: useCase, messages: messages}
}

type RejectSubmissionHandler struct {
	useCase  UseCase
	messages submissionmessage.Messages
}

func (h RejectSubmissionHandler) RejectSubmission(
	ctx context.Context,
	req *connect.Request[flagsv1.RejectSubmissionRequest],
) (*connect.Response[flagsv1.RejectSubmissionResponse], error) {
	rejected, err := h.useCase.Execute(ctx, reject_submission_usecase.In{
		Submission: req.Msg.GetSubmissionId(), Reason: submissionmessage.ReasonOf(req.Msg.GetRejection().GetReason()),
		Note: req.Msg.GetRejection().GetNote(),
	})
	switch {
	case errors.Is(err, submissions.ErrUnknownSubmission):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, submissions.ErrDecided):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, submissions.ErrInvalidRejection):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	kept, err := h.messages.Kept(rejected)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.RejectSubmissionResponse{Submission: kept}), nil
}
