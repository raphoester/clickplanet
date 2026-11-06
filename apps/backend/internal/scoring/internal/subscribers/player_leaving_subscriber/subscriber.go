package player_leaving_subscriber

import (
	"context"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, account standings.AccountID, decision departures.Decision) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*playerv1.PlayerLeaving] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.PlayerLeaving) error {
	account, err := standings.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the event's subject: the sentinel already says what is wrong.
	}
	var decidedAt time.Time
	if event.GetDecidedAt().IsValid() {
		decidedAt = event.GetDecidedAt().AsTime()
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account, departures.Leaves(decidedAt)) //nolint:wrapcheck // the use case named it.
}
