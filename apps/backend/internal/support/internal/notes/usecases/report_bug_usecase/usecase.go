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

type Uploads interface {
	UploadURLs(note notes.Note) ([]string, error)
}

type In struct {
	Account     notes.AccountID
	Text        string
	Diagnostics notes.Diagnostics
	Attachments []notes.Attachment
}

type Out struct {
	// One per attachment of In, in its order.
	UploadURLs []string
}

const writeTimeout = 5 * time.Second

func New(ids notes.IDProvider, clock cptime.Clock, rules notes.AttachmentRules, saver Saver, uploads Uploads) *UseCase {
	return &UseCase{ids: ids, clock: clock, rules: rules, saver: saver, uploads: uploads}
}

type UseCase struct {
	ids     notes.IDProvider
	clock   cptime.Clock
	rules   notes.AttachmentRules
	saver   Saver
	uploads Uploads
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	id, err := u.ids.NewID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw a note id: %w", err)
	}

	note, err := notes.NewBugReport(id, in.Account, in.Text, in.Diagnostics, u.clock.Now())
	if err != nil {
		return Out{}, fmt.Errorf("cannot take the bug report: %w", err)
	}
	note, err = note.Attaching(in.Attachments, u.rules)
	if err != nil {
		return Out{}, fmt.Errorf("cannot take the files of the bug report: %w", err)
	}

	links, err := u.uploads.UploadURLs(note)
	if err != nil {
		return Out{}, fmt.Errorf("failed to make the upload links of the bug report: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := u.saver.Save(ctx, note); err != nil {
		return Out{}, fmt.Errorf("failed to keep the bug report: %w", err)
	}
	return Out{UploadURLs: links}, nil
}
