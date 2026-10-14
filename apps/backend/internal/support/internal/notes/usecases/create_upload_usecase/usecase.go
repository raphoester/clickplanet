package create_upload_usecase

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

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

func New(ids notes.AttachmentIDProvider, rules notes.AttachmentRules, uploads Uploads) *UseCase {
	return &UseCase{ids: ids, rules: rules, uploads: uploads}
}

type UseCase struct {
	ids     notes.AttachmentIDProvider
	rules   notes.AttachmentRules
	uploads Uploads
}

func (u *UseCase) Execute(in In) (Out, error) {
	id, err := u.ids.NewAttachmentID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw an attachment id: %w", err)
	}

	file := notes.Attachment{ID: id, MediaType: in.MediaType, Size: in.Size}
	if err := u.rules.UploadError(file); err != nil {
		return Out{}, fmt.Errorf("cannot take the file: %w", err)
	}

	link, err := u.uploads.UploadURL(file)
	if err != nil {
		return Out{}, fmt.Errorf("failed to make the upload link: %w", err)
	}
	return Out{ID: id, URL: link}, nil
}
