package report_place_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/usecases/report_place_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/caller"
)

type UseCase interface {
	Execute(ctx context.Context, in report_place_usecase.In) error
}

func New(useCase UseCase) ReportPlaceHandler {
	return ReportPlaceHandler{useCase: useCase}
}

type ReportPlaceHandler struct {
	useCase UseCase
}

var reasons = map[supportv1.PlaceReportReason]places.Reason{
	supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE:   places.ReasonHate,
	supportv1.PlaceReportReason_PLACE_REPORT_REASON_SEXUAL: places.ReasonSexual,
	supportv1.PlaceReportReason_PLACE_REPORT_REASON_OTHER:  places.ReasonOther,
}

func (h ReportPlaceHandler) ReportPlace(
	ctx context.Context,
	req *connect.Request[supportv1.ReportPlaceRequest],
) (*connect.Response[supportv1.ReportPlaceResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	err = h.useCase.Execute(ctx, report_place_usecase.In{
		Reporter:  account,
		Tile:      places.TileID(req.Msg.GetTileId()),
		Complaint: places.Complaint{Reason: reasons[req.Msg.GetReason()], Text: req.Msg.GetText()},
	})
	switch {
	case errors.Is(err, places.ErrUnknownReason):
		return nil, connect.NewError(connect.CodeInvalidArgument, places.ErrUnknownReason)
	case errors.Is(err, notes.ErrInvalidText):
		return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidText)
	case errors.Is(err, places.ErrInvalidReport):
		return nil, connect.NewError(connect.CodeInvalidArgument, places.ErrInvalidReport)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&supportv1.ReportPlaceResponse{}), nil
}
