package forget_notes_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	SentBy(ctx context.Context, account notes.AccountID) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

func New(kept Notes) *UseCase {
	return &UseCase{notes: kept}
}

type UseCase struct {
	notes Notes
}

func (u *UseCase) Execute(ctx context.Context, account notes.AccountID) error {
	sent, err := u.notes.SentBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the notes the account sent: %w", err)
	}

	if err := u.notes.Save(ctx, notes.AllForgotten(sent)...); err != nil {
		return fmt.Errorf("failed to forget the account's notes: %w", err)
	}
	return nil
}
