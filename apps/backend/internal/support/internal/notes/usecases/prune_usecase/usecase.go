package prune_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	WithSenderBefore(ctx context.Context, cutoff time.Time, limit int) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

type Executor interface {
	Execute(ctx context.Context) (int64, error)
}

const (
	timeout = 30 * time.Second
	batch   = 500
)

func New(retention time.Duration, clock cptime.Clock, kept Notes) *UseCase {
	return &UseCase{retention: retention, clock: clock, notes: kept}
}

type UseCase struct {
	retention time.Duration
	clock     cptime.Clock
	notes     Notes
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

		if err := u.notes.Save(ctx, notes.AllAnonymized(old)...); err != nil {
			return anonymized, fmt.Errorf("failed to anonymize the notes past retention: %w", err)
		}
		anonymized += int64(len(old))
	}
}
