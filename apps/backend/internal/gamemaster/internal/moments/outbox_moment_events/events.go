package outbox_moment_events

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/momentmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ moments.Events = Events{}

func New(outbox cpatomic.Outbox) Events {
	return Events{outbox: outbox}
}

// Events records each moment as its gamemaster.v1 event, in the transaction ctx carries.
type Events struct {
	outbox cpatomic.Outbox
}

func (e Events) Record(ctx context.Context, moment moments.Moment) error {
	event, err := momentmessage.EventOf(moment)
	if err != nil {
		return err //nolint:wrapcheck // it names the moment.
	}
	if err := e.outbox.Record(ctx, event); err != nil {
		return fmt.Errorf("failed to record %s: %w", event.ProtoReflect().Descriptor().FullName(), err)
	}
	return nil
}
