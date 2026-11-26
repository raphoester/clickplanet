package get_payable_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/get_payable_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type UseCase interface {
	Execute(ctx context.Context, in get_payable_usecase.In) ([]grants.Target, error)
}

func New(useCase UseCase) GetPayableHandler {
	return GetPayableHandler{useCase: useCase}
}

type GetPayableHandler struct {
	useCase UseCase
}

func (h GetPayableHandler) GetPayable(
	ctx context.Context,
	req *connect.Request[flagsv1.GetPayableRequest],
) (*connect.Response[flagsv1.GetPayableResponse], error) {
	target, err := grants.ParseTarget(req.Msg.GetTarget().GetSubmissionId(), req.Msg.GetTarget().GetFlag())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	targets, err := h.useCase.Execute(ctx, get_payable_usecase.In{Account: req.Msg.GetAccountId(), Target: target})
	switch {
	case errors.Is(err, catalog.ErrInvalidAccount):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, submissions.ErrUnknownSubmission), errors.Is(err, catalog.ErrUnknownFlag):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, submissions.ErrNotYours), errors.Is(err, catalog.ErrHeld):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, submissions.ErrNotPayable), errors.Is(err, submissions.ErrPaidAlready),
		errors.Is(err, catalog.ErrRemoved):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	same := make([]*flagsv1.TimeTarget, 0, len(targets))
	for _, target := range targets {
		if id, ok := target.Flag(); ok {
			same = append(same, &flagsv1.TimeTarget{Target: &flagsv1.TimeTarget_Flag{Flag: id.String()}})
			continue
		}
		id, _ := target.Submission()
		same = append(same, &flagsv1.TimeTarget{Target: &flagsv1.TimeTarget_SubmissionId{SubmissionId: id.String()}})
	}
	return connect.NewResponse(&flagsv1.GetPayableResponse{SameFlag: same}), nil
}
