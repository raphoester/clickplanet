package redis_outbox_sender

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
)

type Sender interface {
	Send(ctx context.Context, messages ...proto.Message) error
}

func New(events, commands Sender) Outbox {
	return Outbox{events: events, commands: commands}
}

// What the chat's outbox relay sends with: support's commands to their handler, the chat's events to every listener.
type Outbox struct {
	events   Sender
	commands Sender
}

var errUnknown = errors.New("the chat's outbox keeps no such message")

func (o Outbox) Send(ctx context.Context, messages ...proto.Message) error {
	var events, commands []proto.Message
	for _, message := range messages {
		switch message.(type) {
		case *supportv1.DecideReport:
			commands = append(commands, message)
		case *chatv1.AccountMuted:
			events = append(events, message)
		default:
			return fmt.Errorf("%w: %s", errUnknown, message.ProtoReflect().Descriptor().FullName())
		}
	}

	if err := o.events.Send(ctx, events...); err != nil {
		return fmt.Errorf("failed to tell the judge's mutes: %w", err)
	}
	if err := o.commands.Send(ctx, commands...); err != nil {
		return fmt.Errorf("failed to send the judge's decisions: %w", err)
	}
	return nil
}
