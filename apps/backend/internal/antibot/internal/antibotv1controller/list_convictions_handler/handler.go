package list_convictions_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var errInvalidWindow = errors.New("convictions are read from a since before their until")

type Guard interface {
	Convictions(ctx context.Context, since, until time.Time) ([]guard.Conviction, error)
}

func New(guard Guard, clock cptime.Clock) ListConvictionsHandler {
	return ListConvictionsHandler{guard: guard, clock: clock}
}

type ListConvictionsHandler struct {
	guard Guard
	clock cptime.Clock
}

func (h ListConvictionsHandler) ListConvictions(
	ctx context.Context,
	req *connect.Request[antibotv1.ListConvictionsRequest],
) (*connect.Response[antibotv1.ListConvictionsResponse], error) {
	until := h.clock.Now()
	if at := req.Msg.GetUntil(); at != nil {
		until = at.AsTime()
	}
	since := req.Msg.GetSince()
	if since == nil || !since.AsTime().Before(until) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWindow)
	}

	listed, err := h.guard.Convictions(ctx, since.AsTime(), until)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := &antibotv1.ListConvictionsResponse{Convictions: make([]*antibotv1.Conviction, 0, len(listed))}
	for _, conviction := range listed {
		res.Convictions = append(res.Convictions, encode(conviction))
	}

	return connect.NewResponse(res), nil
}

func encode(conviction guard.Conviction) *antibotv1.Conviction {
	readings := make([]*antibotv1.WatchdogReading, 0, len(conviction.Readings))
	for _, reading := range conviction.Readings {
		readings = append(readings, &antibotv1.WatchdogReading{
			Watchdog: reading.Watchdog,
			Level:    reading.Level,
			Evidence: reading.Evidence,
			At:       timestampOrNil(reading.At),
		})
	}

	return &antibotv1.Conviction{
		At:                 timestamppb.New(conviction.At),
		Scope:              conviction.Scope,
		AccountId:          conviction.Account,
		AlongsideAccountId: conviction.Alongside,
		Flags:              uint32(conviction.Flags),   //nolint:gosec // a flag count, never negative.
		Offence:            uint32(conviction.Offence), //nolint:gosec // an offence count, never negative.
		BannedUntil:        timestamppb.New(conviction.BannedUntil),
		Readings:           readings,
		Clicks:             uint32(conviction.Clicks), //nolint:gosec // a click count, never negative.
		ActiveFor:          durationpb.New(conviction.ActiveFor),
		LongestGap:         durationpb.New(conviction.LongestGap),
		TopCountry:         conviction.TopCountry,
		TopCountryClicks:   uint32(conviction.TopCountryClicks), //nolint:gosec // a click count, never negative.
	}
}

func timestampOrNil(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
