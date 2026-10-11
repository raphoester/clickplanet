package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const DeletedName = "[deleted]"

var deleted = AuthorOf(DeletedName, false, 0, 0, Title{})

type Shown interface {
	Message(ctx context.Context, id MessageID, since time.Time, limit int) (Message, error)
}

type Namer interface {
	Authors(ctx context.Context, accounts []AccountID) (map[AccountID]Author, error)
}

const threadTimeout = 5 * time.Second

func NewThread(shown Shown, namer Namer, clock cptime.Clock, window Window) Thread {
	return Thread{shown: shown, namer: namer, clock: clock, window: window}
}

type Thread struct {
	shown  Shown
	namer  Namer
	clock  cptime.Clock
	window Window
}

func (t Thread) Reply(ctx context.Context, draft Message, replyTo MessageID) (Message, error) {
	if replyTo == "" {
		return draft, nil
	}

	ctx, cancel := context.WithTimeout(ctx, threadTimeout)
	defer cancel()

	answered, err := t.shown.Message(ctx, replyTo, t.window.Since(t.clock.Now()), t.window.Size())
	if err != nil {
		return Message{}, fmt.Errorf("failed to read the message replied to: %w", err)
	}
	named, err := t.named(ctx, answered)
	if err != nil {
		return Message{}, err
	}
	return draft.Answering(named), nil
}

func (t Thread) named(ctx context.Context, message Message) (Message, error) {
	if message.account == NoAccount {
		return message, nil
	}

	authors, err := t.namer.Authors(ctx, []AccountID{message.account})
	if err != nil {
		return Message{}, fmt.Errorf("%w: %w", ErrAuthorUnavailable, err)
	}
	author, known := authors[message.account]
	if !known {
		return message.Named(deleted), nil
	}
	return message.Named(author), nil
}
