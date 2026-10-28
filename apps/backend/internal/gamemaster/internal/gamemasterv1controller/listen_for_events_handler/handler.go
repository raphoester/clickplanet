package listen_for_events_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

type Turns interface {
	Subscribe(ctx context.Context) <-chan *gamemasterv1.GetTurnResponse
}

// Well under Cloudflare's ~125s idle cut on a silent stream.
const DefaultHeartbeat = 30 * time.Second

func New(turns Turns, heartbeat time.Duration) ListenForEventsHandler {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return ListenForEventsHandler{turns: turns, heartbeat: heartbeat}
}

type ListenForEventsHandler struct {
	turns     Turns
	heartbeat time.Duration
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[gamemasterv1.ListenForEventsRequest],
	stream *connect.ServerStream[gamemasterv1.GamemasterEvent],
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	turns := h.turns.Subscribe(ctx)

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	var sent *gamemasterv1.GetTurnResponse

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(&gamemasterv1.GamemasterEvent{
				Event: &gamemasterv1.GamemasterEvent_Heartbeat{Heartbeat: &gamemasterv1.Heartbeat{}},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case now, open := <-turns:
			if !open {
				return nil
			}
			if err := sendChanges(stream, sent, now); err != nil {
				return err
			}
			sent = now
		}
	}
}

// The turn, the timeline, then the debrief, each only when it differs from the one sent last; no debrief before any.
func sendChanges(
	stream *connect.ServerStream[gamemasterv1.GamemasterEvent],
	sent, now *gamemasterv1.GetTurnResponse,
) error {
	if sent == nil || !proto.Equal(sent.GetTurn(), now.GetTurn()) {
		if err := stream.Send(&gamemasterv1.GamemasterEvent{
			Event: &gamemasterv1.GamemasterEvent_Turn{Turn: now.GetTurn()},
		}); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
	}
	if sent == nil || !proto.Equal(sent.GetTimeline(), now.GetTimeline()) {
		if err := stream.Send(&gamemasterv1.GamemasterEvent{
			Event: &gamemasterv1.GamemasterEvent_Timeline{Timeline: now.GetTimeline()},
		}); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
	}
	if now.GetDebrief() != nil && (sent == nil || !proto.Equal(sent.GetDebrief(), now.GetDebrief())) {
		if err := stream.Send(&gamemasterv1.GamemasterEvent{
			Event: &gamemasterv1.GamemasterEvent_Debrief{Debrief: now.GetDebrief()},
		}); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
	}
	return nil
}
