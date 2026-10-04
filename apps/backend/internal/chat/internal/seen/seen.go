package seen

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var ErrNoTime = errors.New("a seen mark needs a time")

// Never later than now: a client clock running ahead would hide what is said next.
func Until(at, now time.Time) (time.Time, error) {
	if !at.After(time.UnixMilli(0)) {
		return time.Time{}, ErrNoTime
	}
	if at.After(now) {
		return now, nil
	}
	return at, nil
}

type Storage interface {
	SaveSeen(ctx context.Context, account messages.AccountID, until time.Time) error
	DeleteSeen(ctx context.Context, account messages.AccountID) error
}
