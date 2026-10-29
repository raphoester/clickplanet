package redis_outbox_sender

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
)

type Sender interface {
	Send(ctx context.Context, messages ...proto.Message) error
}

func New(events, commands Sender) Outbox {
	return Outbox{events: events, commands: commands}
}

// Outbox is what missions' outbox relay sends with: planet's and scoring's commands to their handler, missions' events
// to every listener.
type Outbox struct {
	events   Sender
	commands Sender
}

var errUnknown = errors.New("missions' outbox keeps no such message")

func (o Outbox) Send(ctx context.Context, messages ...proto.Message) error {
	var events, commands []proto.Message
	for _, message := range messages {
		switch message.(type) {
		case *planetv1.GrantCharges, *scoringv1.GrantPoints:
			commands = append(commands, message)
		case *missionsv1.MissionsPaid:
			events = append(events, message)
		default:
			return fmt.Errorf("%w: %s", errUnknown, message.ProtoReflect().Descriptor().FullName())
		}
	}

	if err := o.commands.Send(ctx, commands...); err != nil {
		return fmt.Errorf("failed to send the grants: %w", err)
	}
	if err := o.events.Send(ctx, events...); err != nil {
		return fmt.Errorf("failed to tell the missions paid: %w", err)
	}
	return nil
}
