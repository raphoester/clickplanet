package remove_message_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Messages interface {
	Kept(ctx context.Context, id messages.MessageID, since time.Time) (bool, error)
}

type Saver interface {
	Save(ctx context.Context, removal removals.Snapshot) error
}

type Publisher interface {
	Publish(update feed.Update)
}

func New(messages Messages, saver Saver, publisher Publisher, window messages.Window, clock cptime.Clock) *UseCase {
	return &UseCase{messages: messages, saver: saver, publisher: publisher, window: window, clock: clock}
}

type UseCase struct {
	messages  Messages
	saver     Saver
	publisher Publisher
	window    messages.Window
	clock     cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, id messages.MessageID) (time.Time, error) {
	now := u.clock.Now()

	kept, err := u.messages.Kept(ctx, id, u.window.Since(now))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to find the message: %w", err)
	}
	if !kept {
		return time.Time{}, fmt.Errorf("%w: %q", removals.ErrUnknownMessage, id)
	}

	removal := removals.NewRemoval(id, now)
	if err := u.saver.Save(ctx, removal.TakeSnapshot()); err != nil {
		return time.Time{}, fmt.Errorf("failed to keep the removal: %w", err)
	}
	u.publisher.Publish(feed.MessageRemoved(id))

	return now, nil
}
