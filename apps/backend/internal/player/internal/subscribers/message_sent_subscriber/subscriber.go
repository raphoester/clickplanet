package message_sent_subscriber

import (
	"context"
	"errors"
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_stats_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*chatv1.MessageSent] = Subscriber{}

var errNoTime = errors.New("the message has no time")

func (s Subscriber) Handle(ctx context.Context, event *chatv1.MessageSent) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("message %s: %w", event.GetMessageId(), err)
	}

	if err := event.GetSentAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_stats_usecase.In{Account: account, Change: players.MessageAt(event.GetSentAt().AsTime())}) //nolint:wrapcheck // the use case named it.
}
