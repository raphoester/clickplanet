package report_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/report_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, in report_player_usecase.In) error
}

func New(useCase UseCase) ReportPlayerHandler {
	return ReportPlayerHandler{useCase: useCase}
}

type ReportPlayerHandler struct {
	useCase UseCase
}

var reasons = map[supportv1.ReportReason]reports.Reason{
	supportv1.ReportReason_REPORT_REASON_CHEATING:   reports.ReasonCheating,
	supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE: reports.ReasonChatAbuse,
	supportv1.ReportReason_REPORT_REASON_BAD_NAME:   reports.ReasonBadName,
	supportv1.ReportReason_REPORT_REASON_OTHER:      reports.ReasonOther,
}

func (h ReportPlayerHandler) ReportPlayer(
	ctx context.Context,
	req *connect.Request[supportv1.ReportPlayerRequest],
) (*connect.Response[supportv1.ReportPlayerResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, report_player_usecase.In{
		Reporter: account,
		Name:     req.Msg.GetName(),
		Complaint: reports.Complaint{
			Reason:  reasons[req.Msg.GetReason()],
			Text:    req.Msg.GetText(),
			Message: reports.MessageID(req.Msg.GetMessageId()),
		},
	})
	switch {
	case errors.Is(err, reports.ErrUnknownReason):
		return nil, connect.NewError(connect.CodeInvalidArgument, reports.ErrUnknownReason)
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case errors.Is(err, reports.ErrOwnAccount):
		return nil, connect.NewError(connect.CodeInvalidArgument, reports.ErrOwnAccount)
	case errors.Is(err, reports.ErrNoPlayer):
		return nil, connect.NewError(connect.CodeNotFound, reports.ErrNoPlayer)
	case errors.Is(err, reports.ErrReportedAlready):
		return nil, connect.NewError(connect.CodeAlreadyExists, reports.ErrReportedAlready)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&supportv1.ReportPlayerResponse{}), nil
}
