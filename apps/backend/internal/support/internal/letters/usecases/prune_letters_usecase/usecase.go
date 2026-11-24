package prune_letters_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
)

const timeout = 30 * time.Second

func New(retention time.Duration, clock cptime.Clock, kept letters.Storage) *UseCase {
	return &UseCase{retention: retention, clock: clock, letters: kept}
}

// UseCase anonymizes the letters first written past the retention, as the notes are.
type UseCase struct {
	retention time.Duration
	clock     cptime.Clock
	letters   letters.Storage
}

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var pruned int64
	cutoff := u.clock.Now().Add(-u.retention)
	for {
		old, err := u.letters.WrittenBefore(ctx, cutoff)
		if err != nil {
			return pruned, fmt.Errorf("failed to read the letters past retention: %w", err)
		}
		if len(old) == 0 {
			return pruned, nil
		}
		anonymized := make([]letters.Letter, 0, len(old))
		for _, letter := range old {
			anonymized = append(anonymized, letter.Anonymized())
		}
		if err := u.letters.Save(ctx, anonymized...); err != nil {
			return pruned, fmt.Errorf("failed to anonymize the letters past retention: %w", err)
		}
		pruned += int64(len(anonymized))
	}
}
