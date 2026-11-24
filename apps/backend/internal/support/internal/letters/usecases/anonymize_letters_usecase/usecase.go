package anonymize_letters_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(kept letters.Storage) *UseCase {
	return &UseCase{letters: kept}
}

// UseCase no longer names a deleted account in the letters written to it. What was said stays.
type UseCase struct {
	letters letters.Storage
}

func (u *UseCase) Execute(ctx context.Context, account notes.AccountID) error {
	written, err := u.letters.To(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the letters to the account: %w", err)
	}
	anonymized := make([]letters.Letter, 0, len(written))
	for _, letter := range written {
		anonymized = append(anonymized, letter.Anonymized())
	}
	if err := u.letters.Save(ctx, anonymized...); err != nil {
		return fmt.Errorf("failed to anonymize the letters to the account: %w", err)
	}
	return nil
}
