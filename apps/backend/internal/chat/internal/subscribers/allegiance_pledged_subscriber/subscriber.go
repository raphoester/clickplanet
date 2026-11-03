package allegiance_pledged_subscriber

import (
	"context"
	"errors"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/leave_room_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(in leave_room_usecase.In)
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.AllegiancePledged] = Subscriber{}

var errNoAccount = errors.New("the pledge names no account")

// A first flag leaves no room.
func (s Subscriber) Handle(_ context.Context, event *planetv1.AllegiancePledged) error {
	account := messages.AccountIDOf(event.GetAccountId())
	if account == messages.NoAccount {
		return errNoAccount
	}
	if event.GetPrevious() == "" {
		return nil
	}

	s.useCase.Execute(leave_room_usecase.In{Account: account, Left: messages.FlagRoom(event.GetPrevious())})
	return nil
}
