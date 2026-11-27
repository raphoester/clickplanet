package appeal_submission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/appeal_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in appeal_submission_usecase.In) (appeal_submission_usecase.Out, error)
}

func New(useCase UseCase, messages submissionmessage.Messages) AppealSubmissionHandler {
	return AppealSubmissionHandler{useCase: useCase, messages: messages}
}

type AppealSubmissionHandler struct {
	useCase  UseCase
	messages submissionmessage.Messages
}

func (h AppealSubmissionHandler) AppealSubmission(
	ctx context.Context,
	req *connect.Request[flagsv1.AppealSubmissionRequest],
) (*connect.Response[flagsv1.AppealSubmissionResponse], error) {
	out, err := h.useCase.Execute(ctx, appeal_submission_usecase.In{
		Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx), Submission: req.Msg.GetSubmissionId(),
	})
	switch {
	case errors.Is(err, submissions.ErrGuest), errors.Is(err, submissions.ErrNotYours):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, submissions.ErrWaiting):
		return nil, connect.NewError(connect.CodeAlreadyExists, submissions.ErrWaiting)
	case errors.Is(err, submissions.ErrNotRejected), errors.Is(err, submissions.ErrPictureGone),
		errors.Is(err, appeals.ErrNotTheJudges), errors.Is(err, appeals.ErrAskedAlready):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, submissions.ErrUnknownSubmission):
		return nil, connect.NewError(connect.CodeNotFound, submissions.ErrUnknownSubmission)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.AppealSubmissionResponse{
		Submission: h.messages.Submission(out.Submission, submissionmessage.Reviews{Appeal: out.Appeal}),
	}), nil
}
