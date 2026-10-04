package publishing_count_takes

import (
	"context"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/count_takes_usecase"
)

type Publisher interface {
	Publish(event proto.Message)
}

func New(inner count_takes_usecase.Executor, events Publisher) *Decorator {
	return &Decorator{inner: inner, events: events}
}

type Decorator struct {
	inner  count_takes_usecase.Executor
	events Publisher
}

var _ count_takes_usecase.Executor = (*Decorator)(nil)

func (d *Decorator) Execute(ctx context.Context) (count_takes_usecase.Out, error) {
	out, err := d.inner.Execute(ctx)

	for _, account := range out.Counted {
		d.events.Publish(&playerv1.StatsChanged{AccountId: account.String()})
	}

	return out, err //nolint:wrapcheck // a decorator adds an event, not a sentence.
}
