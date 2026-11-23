package anonymize_notes_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	SentBy(ctx context.Context, account notes.AccountID) ([]notes.Note, error)
	Save(ctx context.Context, notes ...notes.Note) error
}

type Conversations interface {
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]conversations.Conversation, error)
	Save(ctx context.Context, conversations ...conversations.Conversation) error
}

func New(kept Notes, said Conversations) *UseCase {
	return &UseCase{notes: kept, conversations: said}
}

type UseCase struct {
	notes         Notes
	conversations Conversations
}

func (u *UseCase) Execute(ctx context.Context, account notes.AccountID) error {
	sent, err := u.notes.SentBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the notes the account sent: %w", err)
	}

	// The conversations first: once the notes lose their sender, nothing finds them again.
	said, err := u.conversations.Conversations(ctx, notes.IDsOf(sent)...)
	if err != nil {
		return fmt.Errorf("failed to read the conversations of the notes the account sent: %w", err)
	}
	if err := u.conversations.Save(ctx, conversations.AllAnonymized(said)...); err != nil {
		return fmt.Errorf("failed to anonymize the conversations of the notes the account sent: %w", err)
	}

	if err := u.notes.Save(ctx, notes.AllAnonymized(sent)...); err != nil {
		return fmt.Errorf("failed to anonymize the notes the account sent: %w", err)
	}
	return nil
}
