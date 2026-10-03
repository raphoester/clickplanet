package publishing_record_message

import (
	"context"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in record_message_usecase.In) error
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

func (d *Decorator) Execute(ctx context.Context, in record_message_usecase.In) error {
	if err := d.implementation.Execute(ctx, in); err != nil {
		return err //nolint:wrapcheck // a decorator adds an event, not a sentence.
	}

	d.events.Publish(&playerv1.StatsChanged{AccountId: in.Account.String()})
	return nil
}
