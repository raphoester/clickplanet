package decide_place_report_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/reportoutcome"
)

type UseCase interface {
	Execute(ctx context.Context, in decide_report_usecase.In[places.ReportID]) (time.Time, error)
}

func New(useCase UseCase) DecidePlaceReportHandler {
	return DecidePlaceReportHandler{useCase: useCase}
}

type DecidePlaceReportHandler struct {
	useCase UseCase
}

func (h DecidePlaceReportHandler) DecidePlaceReport(
	ctx context.Context,
	req *connect.Request[supportv1.DecidePlaceReportRequest],
) (*connect.Response[supportv1.DecidePlaceReportResponse], error) {
	report, err := places.ReportIDOf(req.Msg.GetReportId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, places.ErrInvalidReportID)
	}

	decidedAt, err := h.useCase.Execute(ctx, decide_report_usecase.In[places.ReportID]{
		Report: report, Outcome: reportoutcome.Of(req.Msg.GetOutcome()), Comment: req.Msg.GetComment(),
	})
	switch {
	case errors.Is(err, decisions.ErrUnknownOutcome):
		return nil, connect.NewError(connect.CodeInvalidArgument, decisions.ErrUnknownOutcome)
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case errors.Is(err, decisions.ErrUnknownReport):
		return nil, connect.NewError(connect.CodeNotFound, decisions.ErrUnknownReport)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&supportv1.DecidePlaceReportResponse{DecidedAt: timestamppb.New(decidedAt)}), nil
}
