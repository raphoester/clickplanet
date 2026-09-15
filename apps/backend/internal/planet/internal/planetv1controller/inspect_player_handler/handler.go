package inspect_player_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/inspect_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in inspect_player_usecase.In) (antibot.Examination, error)
}

func New(useCase UseCase) InspectPlayerHandler {
	return InspectPlayerHandler{useCase: useCase}
}

type InspectPlayerHandler struct {
	useCase UseCase
}

func (h InspectPlayerHandler) InspectPlayer(
	ctx context.Context,
	req *connect.Request[planetv1.InspectPlayerRequest],
) (*connect.Response[planetv1.InspectPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, inspect_player_usecase.In{Scope: req.Msg.GetScope()})

	switch {
	case err == nil:
		return connect.NewResponse(encode(out)), nil
	case errors.Is(err, ledger.ErrInvalidScope):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, inspect_player_usecase.ErrAntiBotOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}

func encode(out antibot.Examination) *planetv1.InspectPlayerResponse {
	readings := make([]*planetv1.WatchdogReading, 0, len(out.Readings))
	for _, reading := range out.Readings {
		readings = append(readings, &planetv1.WatchdogReading{
			Watchdog: reading.Watchdog,
			Level:    reading.Level,
			Evidence: reading.Evidence,
			At:       timestampOrNil(reading.At),
		})
	}

	res := &planetv1.InspectPlayerResponse{
		Scope:            out.Scope,
		Tracked:          &out.Tracked,
		Banned:           &out.Banned,
		BannedUntil:      timestampOrNil(out.BannedUntil),
		Offence:          uint32(out.Offence), //nolint:gosec // an offence count, never negative.
		Flags:            uint32(out.Flags),   //nolint:gosec // a flag count, never negative.
		Readings:         readings,
		Suspects:         uint32(out.Suspects),    //nolint:gosec // bounded by the watchdogs.
		MinSuspects:      uint32(out.MinSuspects), //nolint:gosec // defaulted to at least one.
		Guilty:           &out.Guilty,
		Clicks:           uint32(out.Clicks), //nolint:gosec // a click count, never negative.
		TopCountry:       out.TopCountry,
		TopCountryClicks: uint32(out.TopCountryClicks), //nolint:gosec // a click count, never negative.
		LastClickAt:      timestampOrNil(out.LastClickAt),
	}

	if out.Tracked {
		res.ActiveFor = durationpb.New(out.ActiveFor)
		res.LongestGap = durationpb.New(out.LongestGap)
	}

	return res
}

func timestampOrNil(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
