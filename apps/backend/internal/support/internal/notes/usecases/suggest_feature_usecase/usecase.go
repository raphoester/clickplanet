package suggest_feature_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Saver interface {
	Save(ctx context.Context, notes ...notes.Note) error
}

type Files interface {
	Attachments(ctx context.Context, ids ...notes.AttachmentID) ([]notes.Attachment, error)
}

type In struct {
	Account     notes.AccountID
	Text        string
	Diagnostics notes.Diagnostics
	Attachments []notes.AttachmentID
	// Empty: the address the account holds.
	ReplyAddress cpmail.Address
}

const writeTimeout = 5 * time.Second

func New(
	ids notes.IDProvider, clock cptime.Clock, rules notes.AttachmentRules, saver Saver, files Files, addresses notes.AddressBook,
) *UseCase {
	return &UseCase{ids: ids, clock: clock, rules: rules, saver: saver, files: files, addresses: addresses}
}

type UseCase struct {
	ids       notes.IDProvider
	clock     cptime.Clock
	rules     notes.AttachmentRules
	saver     Saver
	files     Files
	addresses notes.AddressBook
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a note id: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	address, err := notes.ReplyAddress(ctx, in.ReplyAddress, in.Account, u.addresses)
	if err != nil {
		return fmt.Errorf("failed to read where to answer the suggestion: %w", err)
	}

	note, err := notes.NewSuggestion(id, notes.Sent{
		Account: in.Account, Text: in.Text, Diagnostics: in.Diagnostics, ReplyAddress: address,
	}, u.clock.Now())
	if err != nil {
		return fmt.Errorf("cannot take the suggestion: %w", err)
	}

	files, err := u.files.Attachments(ctx, in.Attachments...)
	if err != nil {
		return fmt.Errorf("failed to read the files of the suggestion: %w", err)
	}
	note, err = note.Attaching(files, u.rules)
	if err != nil {
		return fmt.Errorf("cannot take the files of the suggestion: %w", err)
	}

	if err := u.saver.Save(ctx, note); err != nil {
		return fmt.Errorf("failed to keep the suggestion: %w", err)
	}
	return nil
}
