package submit_flag_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/submit_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in submit_flag_usecase.In) (submissions.Snapshot, error)
}

func New(useCase UseCase, messages submissionmessage.Messages) SubmitFlagHandler {
	return SubmitFlagHandler{useCase: useCase, messages: messages}
}

type SubmitFlagHandler struct {
	useCase  UseCase
	messages submissionmessage.Messages
}

func (h SubmitFlagHandler) SubmitFlag(
	ctx context.Context,
	req *connect.Request[flagsv1.SubmitFlagRequest],
) (*connect.Response[flagsv1.SubmitFlagResponse], error) {
	if req.Msg.GetPin() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, catalog.ErrNoPin)
	}
	submission, err := h.useCase.Execute(ctx, submit_flag_usecase.In{
		Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx), Name: req.Msg.GetName(),
		Upload: req.Msg.GetUploadId(), Latitude: req.Msg.GetPin().GetLatitude(),
		Longitude: req.Msg.GetPin().GetLongitude(), Replaces: req.Msg.GetReplaces(),
	})
	switch {
	case errors.Is(err, submissions.ErrGuest):
		return nil, connect.NewError(connect.CodePermissionDenied, submissions.ErrGuest)
	case errors.Is(err, submissions.ErrWaiting):
		return nil, connect.NewError(connect.CodeAlreadyExists, submissions.ErrWaiting)
	case errors.Is(err, submissions.ErrNotWaiting):
		return nil, connect.NewError(connect.CodeFailedPrecondition, submissions.ErrNotWaiting)
	case errors.Is(err, catalog.ErrInvalidName), errors.Is(err, catalog.ErrInvalidPin),
		errors.Is(err, submissions.ErrUnknownSubmission), errors.Is(err, pictures.ErrUnknownUpload),
		errors.Is(err, pictures.ErrTooLarge), errors.Is(err, pictures.ErrNotAPicture),
		errors.Is(err, pictures.ErrMoves), errors.Is(err, pictures.ErrShape), errors.Is(err, pictures.ErrTooSmall):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pictures.ErrOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, pictures.ErrOff)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&flagsv1.SubmitFlagResponse{Submission: h.messages.Submission(submission)}), nil
}
