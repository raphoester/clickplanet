package remove_messages_usecase

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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// A page of ListMessages, at most: one transaction keeps them all.
const MaxRemovals = 200

var (
	ErrNoRemoval       = errors.New("no message to remove")
	ErrTooManyRemovals = fmt.Errorf("more than %d messages to remove at once", MaxRemovals)
	// Kept once, the second reason would be lost from the history without a word.
	ErrRemovedTwice = errors.New("the same message is removed twice")
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

type Removal struct {
	Message messages.MessageID
	Reason  string
}

type In struct {
	Removals []Removal
	By       moderation.Operator
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

type removal struct {
	id      messages.MessageID
	reason  moderation.Reason
	message messages.Message
}

// All or none, as remove_message_usecase removes one: every message is read before any is kept, and they are kept in
// one transaction, so a refusal leaves the chat as it was.
func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	asked, err := check(in.Removals)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot remove the messages: %w", err)
	}

	now := u.clock.Now()
	if err := u.find(ctx, asked, u.window.Since(now)); err != nil {
		return time.Time{}, err
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		for _, removed := range asked {
			if err := u.keep(ctx, removed, now, in.By); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot remove the messages: %w", err)
	}
	for _, removed := range asked {
		u.publisher.Publish(feed.MessageRemoved(removed.id))
	}

	return now, nil
}

func check(asked []Removal) ([]removal, error) {
	if len(asked) == 0 {
		return nil, ErrNoRemoval
	}
	if len(asked) > MaxRemovals {
		return nil, ErrTooManyRemovals
	}

	seen := cpcolls.NewSetWithCapacity[messages.MessageID](len(asked))
	out := make([]removal, 0, len(asked))
	for _, asked := range asked {
		if seen.Contains(asked.Message) {
			return nil, fmt.Errorf("%w: %q", ErrRemovedTwice, asked.Message)
		}
		seen.Add(asked.Message)

		reason, err := moderation.ReasonOf(asked.Reason)
		if err != nil {
			return nil, fmt.Errorf("message %q: %w", asked.Message, err)
		}
		out = append(out, removal{id: asked.Message, reason: reason})
	}
	return out, nil
}

// Fills each removal's message, or refuses them all on the first one past the retention.
func (u *UseCase) find(ctx context.Context, asked []removal, since time.Time) error {
	for i, removed := range asked {
		message, err := u.messages.KeptMessage(ctx, removed.id, since)
		if errors.Is(err, messages.ErrNotKept) {
			return fmt.Errorf("%w: %q", removals.ErrUnknownMessage, removed.id)
		}
		if err != nil {
			return fmt.Errorf("failed to find message %q: %w", removed.id, err)
		}
		asked[i].message = message
	}
	return nil
}

// The history keeps a copy of what the message said, with no address, past the chat's retention.
func (u *UseCase) keep(ctx context.Context, removed removal, now time.Time, by moderation.Operator) error {
	if err := u.saver.Save(ctx, removals.NewRemoval(removed.id, now).TakeSnapshot()); err != nil {
		return fmt.Errorf("failed to keep the removal of message %q: %w", removed.id, err)
	}
	if err := u.history.Record(ctx, moderation.NewRemoval(removed.message, now, removed.reason, by)); err != nil {
		return fmt.Errorf("failed to record the removal of message %q in the moderation history: %w", removed.id, err)
	}
	return nil
}
