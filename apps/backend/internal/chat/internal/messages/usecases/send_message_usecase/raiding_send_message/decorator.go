package raiding_send_message

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
)

type UseCase interface {
	Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error)
}

type RaidMode interface {
	Running(ctx context.Context) (raids.RaidMode, bool, error)
}

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

func New(implementation UseCase, raidMode RaidMode, authors Authors) *Decorator {
	return &Decorator{implementation: implementation, raidMode: raidMode, authors: authors}
}

type Decorator struct {
	implementation UseCase
	raidMode       RaidMode
	authors        Authors
}

// The player module is asked who posts only while raid mode runs: on any other post, the use case asks it once, as
// before.
func (d *Decorator) Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error) {
	if in.Account == messages.NoAccount {
		return d.implementation.Execute(ctx, in) //nolint:wrapcheck // the use case refuses no account.
	}

	mode, running, err := d.raidMode.Running(ctx)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to read whether raid mode runs: %w", err)
	}
	if running {
		author, err := d.authors.Author(ctx, in.Account)
		if err != nil {
			return messages.Message{}, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
		}
		if err := mode.Admits(author); err != nil {
			return messages.Message{}, err //nolint:wrapcheck // the raid mode named it, and the handler maps it.
		}
	}
	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
