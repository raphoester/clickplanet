package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Shown interface {
	Message(ctx context.Context, id MessageID, since time.Time, limit int) (Message, error)
}

const threadTimeout = 5 * time.Second

func NewThread(shown Shown, clock cptime.Clock, window Window) Thread {
	return Thread{shown: shown, clock: clock, window: window}
}

type Thread struct {
	shown  Shown
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
	return draft.Answering(answered)
}
