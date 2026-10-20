package decide_report_handler

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
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/reportoutcome"
)

type UseCase interface {
	Execute(ctx context.Context, in decide_report_usecase.In[reports.ReportID]) (time.Time, error)
}

func New(useCase UseCase) DecideReportHandler {
	return DecideReportHandler{useCase: useCase}
}

type DecideReportHandler struct {
	useCase UseCase
}

func (h DecideReportHandler) DecideReport(
	ctx context.Context,
	req *connect.Request[supportv1.DecideReportRequest],
) (*connect.Response[supportv1.DecideReportResponse], error) {
	report, err := reports.ReportIDOf(req.Msg.GetReportId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, reports.ErrInvalidReportID)
	}

	decidedAt, err := h.useCase.Execute(ctx, decide_report_usecase.In[reports.ReportID]{
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

	return connect.NewResponse(&supportv1.DecideReportResponse{DecidedAt: timestamppb.New(decidedAt)}), nil
}
