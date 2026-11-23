package receive_reply_usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Reader interface {
	// conversations.ErrUnreadable for what is no mail, or has no Message-ID.
	Read(raw io.Reader) (conversations.Received, error)
}

type Notes interface {
	Note(ctx context.Context, id notes.NoteID) (notes.Note, error)
}

type Conversations interface {
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]conversations.Conversation, error)
	Save(ctx context.Context, conversations ...conversations.Conversation) error
}

type In struct {
	// The envelope's: the address the mail was sent to.
	Recipient string
	Raw       io.Reader
}

type Executor interface {
	Execute(ctx context.Context, in In) (notes.NoteID, error)
}

// What is dropped: nothing can be done of it, and the sender is not asked to send it again.
var Dropped = []error{
	conversations.ErrUnreadable, notes.ErrNotANoteAddress, notes.ErrUnknownNote, notes.ErrNoReplyAddress,
	conversations.ErrReceivedAlready,
}

func IsDropped(err error) bool {
	for _, dropped := range Dropped {
		if errors.Is(err, dropped) {
			return true
		}
	}
	return false
}

const timeout = 10 * time.Second

func New(inbox notes.Inbox, reader Reader, kept Notes, said Conversations, clock cptime.Clock) *UseCase {
	return &UseCase{inbox: inbox, reader: reader, notes: kept, conversations: said, clock: clock}
}

type UseCase struct {
	inbox         notes.Inbox
	reader        Reader
	notes         Notes
	conversations Conversations
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, in In) (notes.NoteID, error) {
	note, err := u.inbox.NoteOf(in.Recipient)
	if err != nil {
		return notes.NoteID{}, fmt.Errorf("cannot tell which note the mail answers: %w", err)
	}
	received, err := u.reader.Read(in.Raw)
	if err != nil {
		return note, fmt.Errorf("failed to read the mail: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	answered, err := u.notes.Note(ctx, note)
	if err != nil {
		return note, fmt.Errorf("failed to read the note the mail answers: %w", err)
	}
	if err := answered.AddressError(); err != nil {
		return note, fmt.Errorf("the note takes no reply: %w", err)
	}

	found, err := u.conversations.Conversations(ctx, note)
	if err != nil {
		return note, fmt.Errorf("failed to read the conversation of the note: %w", err)
	}
	replied, err := found[0].Replied(conversations.Reply{
		Mail: received.Mail, From: received.From, Text: u.inbox.Unquoted(received.Text), At: u.clock.Now(),
	})
	if err != nil {
		return note, fmt.Errorf("cannot keep the reply: %w", err)
	}
	if err := u.conversations.Save(ctx, replied); err != nil {
		return note, fmt.Errorf("failed to keep the reply: %w", err)
	}
	return note, nil
}
