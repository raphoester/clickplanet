package open_submitted_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/open_submitted_flag_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in open_submitted_flag_usecase.In) (catalog.Snapshot, error)
}

func New(useCase UseCase, messages flagmessage.Messages) OpenSubmittedFlagHandler {
	return OpenSubmittedFlagHandler{useCase: useCase, messages: messages}
}

type OpenSubmittedFlagHandler struct {
	useCase  UseCase
	messages flagmessage.Messages
}

func (h OpenSubmittedFlagHandler) OpenSubmittedFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.OpenSubmittedFlagRequest],
) (*connect.Response[flagsv1.OpenSubmittedFlagResponse], error) {
	flag, err := h.useCase.Execute(ctx, open_submitted_flag_usecase.In{
		Submission: req.Msg.GetSubmissionId(), EndsAt: req.Msg.GetEndsAt().AsTime(),
	})
	switch {
	case errors.Is(err, submissions.ErrUnknownSubmission):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, submissions.ErrNotApproved), errors.Is(err, catalog.ErrNameTaken):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, catalog.ErrEnded):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.OpenSubmittedFlagResponse{Flag: h.messages.Flag(flag)}), nil
}
