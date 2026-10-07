package publishing_mute

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
)

type Publisher interface {
	Publish(event proto.Message)
}

func New(implementation mute_usecase.Executor, events Publisher) *Decorator {
	return &Decorator{implementation: implementation, events: events}
}

type Decorator struct {
	implementation mute_usecase.Executor
	events         Publisher
}

var _ mute_usecase.Executor = (*Decorator)(nil)

func (d *Decorator) Execute(ctx context.Context, in mute_usecase.In) (mutes.Mute, error) {
	mute, err := d.implementation.Execute(ctx, in)
	if err != nil {
		return mute, err //nolint:wrapcheck // a decorator adds an event, not a sentence.
	}

	d.events.Publish(&chatv1.AccountMuted{
		AccountId: mute.Caller().Account().String(),
		MutedAt:   timestamppb.New(mute.At()),
		Duration:  durationpb.New(mute.Duration()),
	})
	return mute, nil
}
