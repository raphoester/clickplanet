package anonymize_notes_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	SentBy(ctx context.Context, account notes.AccountID) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

type Attachments interface {
	Delete(ctx context.Context, of ...notes.Note) error
}

func New(kept Notes, attachments Attachments) *UseCase {
	return &UseCase{notes: kept, attachments: attachments}
}

type UseCase struct {
	notes       Notes
	attachments Attachments
}

func (u *UseCase) Execute(ctx context.Context, account notes.AccountID) error {
	sent, err := u.notes.SentBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the notes the account sent: %w", err)
	}

	// Before the save: a note that still carries its files is found again, files with no note are not.
	if err := u.attachments.Delete(ctx, sent...); err != nil {
		return fmt.Errorf("failed to delete the files of the notes the account sent: %w", err)
	}

	if err := u.notes.Save(ctx, notes.AllAnonymized(sent)...); err != nil {
		return fmt.Errorf("failed to anonymize the notes the account sent: %w", err)
	}
	return nil
}
