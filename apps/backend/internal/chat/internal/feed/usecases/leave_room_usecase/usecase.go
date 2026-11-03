package leave_room_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Publisher interface {
	Publish(update feed.Update)
}

type In struct {
	Account messages.AccountID
	Left    messages.Room
}

func New(publisher Publisher) *UseCase {
	return &UseCase{publisher: publisher}
}

// Ends the account's streams of the room it left.
type UseCase struct {
	publisher Publisher
}

func (u *UseCase) Execute(in In) {
	u.publisher.Publish(feed.RoomLeft(in.Account, in.Left))
}
