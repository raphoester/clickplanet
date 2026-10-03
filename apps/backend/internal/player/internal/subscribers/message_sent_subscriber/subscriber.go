package message_sent_subscriber

import (
	"context"
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in record_message_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*chatv1.MessageSent] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *chatv1.MessageSent) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("message %s: %w", event.GetMessageId(), err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_message_usecase.In{Account: account}) //nolint:wrapcheck // the use case named it.
}
