package close_letter_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
)

func New(kept letters.Storage, clock cptime.Clock) *UseCase {
	return &UseCase{letters: kept, clock: clock}
}

type UseCase struct {
	letters letters.Storage
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, letter string) (time.Time, error) {
	id, err := letters.LetterIDOf(letter)
	if err != nil {
		return time.Time{}, err //nolint:wrapcheck // the parse names it.
	}
	kept, err := u.letters.Letter(ctx, id)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read the letter to close: %w", err)
	}
	at := u.clock.Now()
	if err := u.letters.Save(ctx, kept.Closed(at)); err != nil {
		return time.Time{}, fmt.Errorf("failed to close the letter: %w", err)
	}
	return at, nil
}
