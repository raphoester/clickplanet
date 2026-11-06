package remove_message_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Messages interface {
	KeptMessage(ctx context.Context, id messages.MessageID, since time.Time) (messages.Message, error)
}

type Saver interface {
	Save(ctx context.Context, removal removals.Snapshot) error
}

type Recorder interface {
	Record(ctx context.Context, action moderation.Action) error
}

type Publisher interface {
	Publish(update feed.Update)
}

type In struct {
	Message messages.MessageID
	Reason  string
	By      moderation.Operator
}

type Executor interface {
	Execute(ctx context.Context, in In) (time.Time, error)
}

func New(
	transactioner cpatomic.Transactioner, messages Messages, saver Saver, history Recorder, publisher Publisher,
	window messages.Window, clock cptime.Clock,
) *UseCase {
	return &UseCase{
		transactioner: transactioner, messages: messages, saver: saver, history: history, publisher: publisher,
		window: window, clock: clock,
	}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	messages      Messages
	saver         Saver
	history       Recorder
	publisher     Publisher
	window        messages.Window
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// The history keeps a copy of what the message said, with no address, past the chat's retention.
func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	reason, err := moderation.ReasonOf(in.Reason)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot remove the message: %w", err)
	}

	now := u.clock.Now()
	message, err := u.messages.KeptMessage(ctx, in.Message, u.window.Since(now))
	if errors.Is(err, messages.ErrNotKept) {
		return time.Time{}, fmt.Errorf("%w: %q", removals.ErrUnknownMessage, in.Message)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to find the message: %w", err)
	}

	removal := removals.NewRemoval(in.Message, now)
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.saver.Save(ctx, removal.TakeSnapshot()); err != nil {
			return fmt.Errorf("failed to keep the removal: %w", err)
		}
		if err := u.history.Record(ctx, moderation.NewRemoval(message, now, reason, in.By)); err != nil {
			return fmt.Errorf("failed to record the removal in the moderation history: %w", err)
		}
		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot remove the message: %w", err)
	}
	u.publisher.Publish(feed.MessageRemoved(in.Message))

	return now, nil
}
