package name_left_subscriber

import (
	"context"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, name pastnames.PastName) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*playerv1.NameLeft] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.NameLeft) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the event's subject: the sentinel already says what is wrong.
	}
	var leftAt time.Time
	if event.GetLeftAt().IsValid() {
		leftAt = event.GetLeftAt().AsTime()
	}
	name, err := pastnames.Snapshot{Account: account, Name: players.Name(event.GetName()), LeftAt: leftAt}.Restore()
	if err != nil {
		return err //nolint:wrapcheck // the domain's sentinel says what the event lacks.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, name) //nolint:wrapcheck // the use case named it.
}
