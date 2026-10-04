package mark_seen_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Saver interface {
	SaveSeen(ctx context.Context, account messages.AccountID, until time.Time) error
}

type In struct {
	Account messages.AccountID
	At      time.Time
}

func New(saver Saver, clock cptime.Clock) *UseCase {
	return &UseCase{saver: saver, clock: clock}
}

type UseCase struct {
	saver Saver
	clock cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if in.Account == messages.NoAccount {
		return messages.ErrNoAccount
	}

	until, err := seen.Until(in.At, u.clock.Now())
	if err != nil {
		return fmt.Errorf("failed to read the seen mark: %w", err)
	}

	if err := u.saver.SaveSeen(ctx, in.Account, until); err != nil {
		return fmt.Errorf("failed to keep what the chat has seen: %w", err)
	}
	return nil
}
