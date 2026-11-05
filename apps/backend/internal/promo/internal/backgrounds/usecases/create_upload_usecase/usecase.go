package create_upload_usecase

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
)

type Uploads interface {
	UploadURL(file backgrounds.File) (string, error)
}

type In struct {
	MediaType backgrounds.MediaType
	Size      int64
}

type Out struct {
	ID  backgrounds.BackgroundID
	URL string
}

func New(ids backgrounds.IDProvider, rules backgrounds.Rules, uploads Uploads) *UseCase {
	return &UseCase{ids: ids, rules: rules, uploads: uploads}
}

type UseCase struct {
	ids     backgrounds.IDProvider
	rules   backgrounds.Rules
	uploads Uploads
}

func (u *UseCase) Execute(in In) (Out, error) {
	id, err := u.ids.NewID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw a background id: %w", err)
	}
	file := backgrounds.File{ID: id, MediaType: in.MediaType, Size: in.Size}
	if err := u.rules.FileError(file); err != nil {
		return Out{}, fmt.Errorf("cannot take the background: %w", err)
	}
	link, err := u.uploads.UploadURL(file)
	if err != nil {
		return Out{}, fmt.Errorf("failed to make the upload link: %w", err)
	}
	return Out{ID: id, URL: link}, nil
}
