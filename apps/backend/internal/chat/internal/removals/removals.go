package removals

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var ErrUnknownMessage = errors.New("no message kept under this id")

// The message row stays, as evidence: a removal only takes it out of what anybody is shown.
type Removal struct {
	message messages.MessageID
	at      time.Time
}

func NewRemoval(message messages.MessageID, at time.Time) Removal {
	return Removal{message: message, at: at}
}

type Snapshot struct {
	MessageID messages.MessageID
	RemovedAt time.Time
}

func (r Removal) TakeSnapshot() Snapshot {
	return Snapshot{MessageID: r.message, RemovedAt: r.at}
}

type Storage interface {
	Save(ctx context.Context, removal Snapshot) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
