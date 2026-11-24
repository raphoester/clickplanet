package receive_letter_reply_usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Reader interface {
	// conversations.ErrUnreadable for what is no mail, or has no Message-ID.
	Read(raw io.Reader) (conversations.Received, error)
}

type In struct {
	// The envelope's: the address the mail was sent to.
	Recipient string
	Raw       io.Reader
}

type Executor interface {
	Execute(ctx context.Context, in In) (letters.LetterID, error)
}

// A stray mail: no letter keeps it, so the operator reads it in their own mailbox.
var Stray = []error{conversations.ErrUnreadable, letters.ErrUnknownLetter, notes.ErrNoReplyAddress}

func IsStray(err error) bool {
	for _, stray := range Stray {
		if errors.Is(err, stray) {
			return true
		}
	}
	return false
}

const timeout = 10 * time.Second

func New(inbox notes.Inbox, reader Reader, kept letters.Storage, clock cptime.Clock) *UseCase {
	return &UseCase{inbox: inbox, reader: reader, letters: kept, clock: clock}
}

// UseCase keeps a player's reply to a letter, the mails it quotes cut off. ErrNotALetter, before reading the mail, for
// a recipient that names no letter.
var _ Executor = (*UseCase)(nil)

type UseCase struct {
	inbox   notes.Inbox
	reader  Reader
	letters letters.Storage
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (letters.LetterID, error) {
	id, err := letters.LetterOf(u.inbox, in.Recipient)
	if err != nil {
		return letters.LetterID{}, fmt.Errorf("cannot tell which letter the mail answers: %w", err)
	}
	received, err := u.reader.Read(in.Raw)
	if err != nil {
		return id, fmt.Errorf("failed to read the mail: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	letter, err := u.letters.Letter(ctx, id)
	if err != nil {
		return id, fmt.Errorf("failed to read the letter the mail answers: %w", err)
	}
	if err := letter.AddressError(); err != nil {
		return id, fmt.Errorf("the letter takes no reply: %w", err)
	}
	replied, err := letter.Replied(conversations.Reply{
		Mail: received.Mail, From: received.From, Text: u.inbox.Unquoted(received.Text), At: u.clock.Now(),
	})
	if err != nil {
		return id, fmt.Errorf("cannot keep the reply: %w", err)
	}
	if err := u.letters.Save(ctx, replied); err != nil {
		return id, fmt.Errorf("failed to keep the reply: %w", err)
	}
	return id, nil
}
