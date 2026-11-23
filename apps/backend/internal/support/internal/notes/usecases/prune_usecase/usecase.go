package prune_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	WithSenderBefore(ctx context.Context, cutoff time.Time, limit int) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

type Conversations interface {
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]conversations.Conversation, error)
	Save(ctx context.Context, conversations ...conversations.Conversation) error
}

type Executor interface {
	Execute(ctx context.Context) (int64, error)
}

const (
	timeout = 30 * time.Second
	batch   = 500
)

func New(retention time.Duration, clock cptime.Clock, kept Notes, said Conversations) *UseCase {
	return &UseCase{retention: retention, clock: clock, notes: kept, conversations: said}
}

type UseCase struct {
	retention     time.Duration
	clock         cptime.Clock
	notes         Notes
	conversations Conversations
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cutoff := u.clock.Now().Add(-u.retention)

	var anonymized int64
	for {
		old, err := u.notes.WithSenderBefore(ctx, cutoff, batch)
		if err != nil {
			return anonymized, fmt.Errorf("failed to read the notes past retention: %w", err)
		}
		if len(old) == 0 {
			return anonymized, nil
		}

		said, err := u.conversations.Conversations(ctx, notes.IDsOf(old)...)
		if err != nil {
			return anonymized, fmt.Errorf("failed to read the conversations of the notes past retention: %w", err)
		}
		if err := u.conversations.Save(ctx, conversations.AllAnonymized(said)...); err != nil {
			return anonymized, fmt.Errorf("failed to anonymize the conversations of the notes past retention: %w", err)
		}

		if err := u.notes.Save(ctx, notes.AllAnonymized(old)...); err != nil {
			return anonymized, fmt.Errorf("failed to anonymize the notes past retention: %w", err)
		}
		anonymized += int64(len(old))
	}
}
