package reopen_submission_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/reopen_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type UseCase interface {
	Execute(ctx context.Context, in reopen_submission_usecase.In) (reopen_submission_usecase.Out, error)
}

func New(useCase UseCase, messages submissionmessage.Messages) ReopenSubmissionHandler {
	return ReopenSubmissionHandler{useCase: useCase, messages: messages}
}

type ReopenSubmissionHandler struct {
	useCase  UseCase
	messages submissionmessage.Messages
}

func (h ReopenSubmissionHandler) ReopenSubmission(
	ctx context.Context,
	req *connect.Request[flagsv1.ReopenSubmissionRequest],
) (*connect.Response[flagsv1.ReopenSubmissionResponse], error) {
	out, err := h.useCase.Execute(ctx, reopen_submission_usecase.In{Submission: req.Msg.GetSubmissionId()})
	switch {
	case errors.Is(err, submissions.ErrUnknownSubmission):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, submissions.ErrNotRejected), errors.Is(err, submissions.ErrPictureGone),
		errors.Is(err, submissions.ErrWaiting), errors.Is(err, appeals.ErrNotTheJudges),
		errors.Is(err, appeals.ErrAskedAlready):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	kept, err := h.messages.Kept(out.Submission)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	kept.Judgement = submissionmessage.Judgement(out.Judgement)
	kept.Appeal = submissionmessage.Appeal(out.Appeal)
	return connect.NewResponse(&flagsv1.ReopenSubmissionResponse{Submission: kept}), nil
}
