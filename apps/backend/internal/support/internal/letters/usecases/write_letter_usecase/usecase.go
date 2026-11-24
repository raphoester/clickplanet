package write_letter_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Post interface {
	// notes.ErrNotSent when the mail did not go.
	Send(ctx context.Context, mail letters.Mail) (cpmail.MessageID, error)
}

type In struct {
	// Empty for a new letter.
	Letter  string
	Account string
	Subject string
	Text    string
}

func New(ids letters.IDProvider, book notes.AddressBook, kept letters.Storage, post Post, clock cptime.Clock) *UseCase {
	return &UseCase{ids: ids, book: book, letters: kept, post: post, clock: clock}
}

type UseCase struct {
	ids     letters.IDProvider
	book    notes.AddressBook
	letters letters.Storage
	post    Post
	clock   cptime.Clock
}

// A mail that did not go keeps nothing; one that went and could not be kept would go twice if sent again.
func (u *UseCase) Execute(ctx context.Context, in In) (letters.Snapshot, error) {
	if in.Letter == "" {
		return u.start(ctx, in)
	}
	id, err := letters.LetterIDOf(in.Letter)
	if err != nil {
		return letters.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	letter, err := u.letters.Letter(ctx, id)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to read the letter: %w", err)
	}
	mail, err := letter.Mail(in.Text)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("cannot write in the letter: %w", err)
	}
	sent, err := u.post.Send(ctx, mail)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to mail the letter: %w", err)
	}
	written, err := letter.Sent(mail.Text, sent, u.clock.Now())
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("cannot keep the mail: %w", err)
	}
	if err := u.letters.Save(ctx, written); err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to keep the mail, which was mailed: %w", err)
	}
	return written.TakeSnapshot(), nil
}

func (u *UseCase) start(ctx context.Context, in In) (letters.Snapshot, error) {
	account, err := notes.AccountIDOf(in.Account)
	if err != nil {
		return letters.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	to, err := u.book.Address(ctx, account)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to read where to write: %w", err)
	}
	id, err := u.ids.NewID()
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to draw a letter id: %w", err)
	}
	draft, err := letters.NewDraft(id, account, to, in.Subject)
	if err != nil {
		return letters.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}
	mail, err := draft.Mail(in.Text)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("cannot write the letter: %w", err)
	}
	sent, err := u.post.Send(ctx, mail)
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to mail the letter: %w", err)
	}
	letter, err := draft.Sent(mail.Text, sent, u.clock.Now())
	if err != nil {
		return letters.Snapshot{}, fmt.Errorf("cannot keep the letter: %w", err)
	}
	if err := u.letters.Save(ctx, letter); err != nil {
		return letters.Snapshot{}, fmt.Errorf("failed to keep the letter, which was mailed: %w", err)
	}
	return letter.TakeSnapshot(), nil
}
