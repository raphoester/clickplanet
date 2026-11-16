package rally_called_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_rally_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_rally_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.RallyCalled] = Subscriber{}

var (
	errNoAccount = errors.New("the rally names no account")
	errNoTime    = errors.New("the rally has no time")
)

func (s Subscriber) Handle(ctx context.Context, event *planetv1.RallyCalled) error {
	rally, err := announcements.RallyIDOf(event.GetRallyId())
	if err != nil {
		return err //nolint:wrapcheck // RallyIDOf named it.
	}
	account := messages.AccountIDOf(event.GetAccountId())
	if account == messages.NoAccount {
		return fmt.Errorf("%w: %q", errNoAccount, event.GetAccountId())
	}
	if err := event.GetCalledAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_rally_usecase.In{ //nolint:wrapcheck // the use case named it.
		Rally:    rally,
		Account:  account,
		Country:  event.GetCountry(),
		Ground:   event.GetGround(),
		Landmass: event.GetLandmassId(),
		At:       event.GetCalledAt().AsTime(),
	})
}
