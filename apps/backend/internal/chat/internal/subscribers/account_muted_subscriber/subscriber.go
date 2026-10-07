package account_muted_subscriber

import (
	"context"
	"errors"
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_mute_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*chatv1.AccountMuted] = Subscriber{}

var (
	errNoAccount  = errors.New("the mute names no account")
	errNoTime     = errors.New("the mute has no time")
	errNoDuration = errors.New("the mute has no duration")
)

func (s Subscriber) Handle(ctx context.Context, event *chatv1.AccountMuted) error {
	account := messages.AccountIDOf(event.GetAccountId())
	if account == messages.NoAccount {
		return fmt.Errorf("%w: %q", errNoAccount, event.GetAccountId())
	}
	if err := event.GetMutedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	if err := event.GetDuration().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoDuration, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_mute_usecase.In{ //nolint:wrapcheck // the use case named it.
		Account:  account,
		At:       event.GetMutedAt().AsTime(),
		Duration: event.GetDuration().AsDuration(),
	})
}
