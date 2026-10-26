package listen_for_events_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
)

type Turns interface {
	Subscribe(ctx context.Context) <-chan *metagamev1.Turn
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
	_ *connect.Request[metagamev1.ListenForEventsRequest],
	stream *connect.ServerStream[metagamev1.MetagameEvent],
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	turns := h.turns.Subscribe(ctx)

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(&metagamev1.MetagameEvent{
				Event: &metagamev1.MetagameEvent_Heartbeat{Heartbeat: &metagamev1.Heartbeat{}},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case turn, open := <-turns:
			if !open {
				return nil
			}
			if err := stream.Send(&metagamev1.MetagameEvent{
				Event: &metagamev1.MetagameEvent_Turn{Turn: turn},
			}); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
