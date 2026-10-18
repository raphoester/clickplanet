package report_bug_usecase

import (
	"context"
	"fmt"
	"time"

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
}

const writeTimeout = 5 * time.Second

func New(ids notes.IDProvider, clock cptime.Clock, rules notes.AttachmentRules, saver Saver, files Files) *UseCase {
	return &UseCase{ids: ids, clock: clock, rules: rules, saver: saver, files: files}
}

type UseCase struct {
	ids   notes.IDProvider
	clock cptime.Clock
	rules notes.AttachmentRules
	saver Saver
	files Files
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a note id: %w", err)
	}

	note, err := notes.NewBugReport(id, in.Account, in.Text, in.Diagnostics, u.clock.Now())
	if err != nil {
		return fmt.Errorf("cannot take the bug report: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	files, err := u.files.Attachments(ctx, in.Attachments...)
	if err != nil {
		return fmt.Errorf("failed to read the files of the bug report: %w", err)
	}
	note, err = note.Attaching(files, u.rules)
	if err != nil {
		return fmt.Errorf("cannot take the files of the bug report: %w", err)
	}

	if err := u.saver.Save(ctx, note); err != nil {
		return fmt.Errorf("failed to keep the bug report: %w", err)
	}
	return nil
}
