package close_note_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Conversations interface {
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]conversations.Conversation, error)
	// notes.ErrUnknownNote for a note no longer kept.
	Save(ctx context.Context, conversations ...conversations.Conversation) error
}

func New(said Conversations, clock cptime.Clock) *UseCase {
	return &UseCase{conversations: said, clock: clock}
}

type UseCase struct {
	conversations Conversations
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, note notes.NoteID) (time.Time, error) {
	found, err := u.conversations.Conversations(ctx, note)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read the conversation of the note: %w", err)
	}

	closedAt := u.clock.Now()
	if err := u.conversations.Save(ctx, found[0].Closed(closedAt)); err != nil {
		return time.Time{}, fmt.Errorf("failed to close the note: %w", err)
	}
	return closedAt, nil
}
