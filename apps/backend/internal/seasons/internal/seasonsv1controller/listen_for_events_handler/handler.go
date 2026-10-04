package listen_for_events_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
)

type Boards interface {
	Subscribe(ctx context.Context, country string) <-chan *seasonsv1.Board
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

var ErrUnknownCountry = errors.New("not a country")

// Well under Cloudflare's ~125s idle cut on a silent stream.
const DefaultHeartbeat = 30 * time.Second

func New(boards Boards, countries CountryChecker, heartbeat time.Duration) ListenForEventsHandler {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return ListenForEventsHandler{boards: boards, countries: countries, heartbeat: heartbeat}
}

type ListenForEventsHandler struct {
	boards    Boards
	countries CountryChecker
	heartbeat time.Duration
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	req *connect.Request[seasonsv1.ListenForEventsRequest],
	stream *connect.ServerStream[seasonsv1.SeasonEvent],
) error {
	country := req.Msg.GetCountryId()
	if country != "" && !h.countries.CheckCountry(country) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %q", ErrUnknownCountry, country))
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	boards := h.boards.Subscribe(ctx, country)

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(&seasonsv1.SeasonEvent{
				Event: &seasonsv1.SeasonEvent_Heartbeat{Heartbeat: &seasonsv1.Heartbeat{}},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case board, open := <-boards:
			if !open {
				return nil
			}
			if err := stream.Send(&seasonsv1.SeasonEvent{
				Event: &seasonsv1.SeasonEvent_Board{Board: board},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
