package create_upload_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

// Keeps a file's room in the bucket before its link is signed: a link cannot be taken back.
type Shelf interface {
	Reserve(ctx context.Context, file notes.Attachment) error
}

type Uploads interface {
	UploadURL(file notes.Attachment) (string, error)
}

type In struct {
	MediaType notes.MediaType
	Size      int64
}

type Out struct {
	ID  notes.AttachmentID
	URL string
}

const writeTimeout = 5 * time.Second

func New(ids notes.AttachmentIDProvider, rules notes.AttachmentRules, shelf Shelf, uploads Uploads) *UseCase {
	return &UseCase{ids: ids, rules: rules, shelf: shelf, uploads: uploads}
}

type UseCase struct {
	ids     notes.AttachmentIDProvider
	rules   notes.AttachmentRules
	shelf   Shelf
	uploads Uploads
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	id, err := u.ids.NewAttachmentID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw an attachment id: %w", err)
	}

	file := notes.Attachment{ID: id, MediaType: in.MediaType, Size: in.Size}
	if err := u.rules.UploadError(file); err != nil {
		return Out{}, fmt.Errorf("cannot take the file: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := u.shelf.Reserve(ctx, file); err != nil {
		return Out{}, fmt.Errorf("cannot keep the file: %w", err)
	}

	link, err := u.uploads.UploadURL(file)
	if err != nil {
		return Out{}, fmt.Errorf("failed to make the upload link: %w", err)
	}
	return Out{ID: id, URL: link}, nil
}
