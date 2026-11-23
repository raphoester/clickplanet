package answer_note_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Notes interface {
	Note(ctx context.Context, id notes.NoteID) (notes.Note, error)
}

type Conversations interface {
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]conversations.Conversation, error)
	Save(ctx context.Context, conversations ...conversations.Conversation) error
}

type Post interface {
	// notes.ErrNotSent when the mail did not go.
	Send(ctx context.Context, letter notes.Letter) (cpmail.MessageID, error)
}

type In struct {
	Note notes.NoteID
	Text string
}

func New(kept Notes, said Conversations, post Post, clock cptime.Clock) *UseCase {
	return &UseCase{notes: kept, conversations: said, post: post, clock: clock}
}

type UseCase struct {
	notes         Notes
	conversations Conversations
	post          Post
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	note, err := u.notes.Note(ctx, in.Note)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read the note to answer: %w", err)
	}
	found, err := u.conversations.Conversations(ctx, in.Note)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read the conversation of the note: %w", err)
	}
	conversation := found[0]

	letter, err := note.Answer(in.Text, conversation.Thread())
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot answer the note: %w", err)
	}
	mail, err := u.post.Send(ctx, letter)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to mail the answer: %w", err)
	}

	sentAt := u.clock.Now()
	answered, err := conversation.Answered(in.Text, mail, sentAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot keep the answer: %w", err)
	}
	if err := u.conversations.Save(ctx, answered); err != nil {
		return time.Time{}, fmt.Errorf("failed to keep the answer, which was mailed: %w", err)
	}
	return sentAt, nil
}
