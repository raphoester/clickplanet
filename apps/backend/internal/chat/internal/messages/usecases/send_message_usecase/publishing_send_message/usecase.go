package publishing_send_message

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error)
}

type Publisher interface {
	Publish(event proto.Message)
}

func New(implementation UseCase, events Publisher) *Decorator {
	return &Decorator{implementation: implementation, events: events}
}

type Decorator struct {
	implementation UseCase
	events         Publisher
}

func (d *Decorator) Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error) {
	message, err := d.implementation.Execute(ctx, in)
	if err != nil {
		return message, err //nolint:wrapcheck // a decorator adds an event, not a sentence.
	}

	d.events.Publish(&chatv1.MessageSent{
		MessageId: string(message.ID()),
		AccountId: message.Account().String(),
		SentAt:    timestamppb.New(message.SentAt()),
	})
	return message, nil
}
