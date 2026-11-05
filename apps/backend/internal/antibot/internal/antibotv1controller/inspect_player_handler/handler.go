package inspect_player_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/inspect_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in inspect_player_usecase.In) (guard.Examination, error)
}

func New(useCase UseCase) InspectPlayerHandler {
	return InspectPlayerHandler{useCase: useCase}
}

type InspectPlayerHandler struct {
	useCase UseCase
}

func (h InspectPlayerHandler) InspectPlayer(
	ctx context.Context,
	req *connect.Request[antibotv1.InspectPlayerRequest],
) (*connect.Response[antibotv1.InspectPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, inspect_player_usecase.In{Scope: req.Msg.GetScope(), Account: req.Msg.GetAccountId()})

	switch {
	case err == nil:
		return connect.NewResponse(encode(out)), nil
	case errors.Is(err, bans.ErrInvalidScope), errors.Is(err, bans.ErrInvalidAccount), errors.Is(err, bans.ErrNoCaller):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}

func encode(out guard.Examination) *antibotv1.InspectPlayerResponse {
	readings := make([]*antibotv1.WatchdogReading, 0, len(out.Readings))
	for _, reading := range out.Readings {
		readings = append(readings, &antibotv1.WatchdogReading{
			Watchdog: reading.Watchdog,
			Level:    reading.Level,
			Evidence: reading.Evidence,
			At:       timestampOrNil(reading.At),
		})
	}

	res := &antibotv1.InspectPlayerResponse{
		Scope:            out.Scope,
		AccountId:        out.Account,
		Tracked:          &out.Tracked,
		Banned:           &out.Banned,
		BannedUntil:      timestampOrNil(out.BannedUntil),
		Offence:          uint32(out.Offence), //nolint:gosec // an offence count, never negative.
		Flags:            uint32(out.Flags),   //nolint:gosec // a flag count, never negative.
		PardonedUntil:    timestampOrNil(out.PardonedUntil),
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
