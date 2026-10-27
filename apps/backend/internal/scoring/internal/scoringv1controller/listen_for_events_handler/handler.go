package listen_for_events_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
)

type Boards interface {
	Subscribe(ctx context.Context, country string) <-chan *scoringv1.Board
}

type Races interface {
	Subscribe(ctx context.Context) <-chan *scoringv1.Race
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

var ErrUnknownCountry = errors.New("not a country")

// Well under Cloudflare's ~125s idle cut on a silent stream.
const DefaultHeartbeat = 30 * time.Second

func New(boards Boards, races Races, countries CountryChecker, heartbeat time.Duration) ListenForEventsHandler {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return ListenForEventsHandler{boards: boards, races: races, countries: countries, heartbeat: heartbeat}
}

type ListenForEventsHandler struct {
	boards    Boards
	races     Races
	countries CountryChecker
	heartbeat time.Duration
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	req *connect.Request[scoringv1.ListenForEventsRequest],
	stream *connect.ServerStream[scoringv1.SeasonEvent],
) error {
	country := req.Msg.GetCountryId()
	if country != "" && !h.countries.CheckCountry(country) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %q", ErrUnknownCountry, country))
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	boards := h.boards.Subscribe(ctx, country)
	races := h.races.Subscribe(ctx)

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(&scoringv1.SeasonEvent{
				Event: &scoringv1.SeasonEvent_Heartbeat{Heartbeat: &scoringv1.Heartbeat{}},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case board, open := <-boards:
			if !open {
				return nil
			}
			if err := stream.Send(&scoringv1.SeasonEvent{
				Event: &scoringv1.SeasonEvent_Board{Board: board},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case race, open := <-races:
			if !open {
				return nil
			}
			if err := stream.Send(&scoringv1.SeasonEvent{
				Event: &scoringv1.SeasonEvent_Race{Race: race},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
