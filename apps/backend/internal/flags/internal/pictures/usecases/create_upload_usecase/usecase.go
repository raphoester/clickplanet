package create_upload_usecase

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

type Bucket interface {
	UploadURL(file pictures.File) (string, error)
}

type In struct {
	ContentType string
	Size        int64
}

type Out struct {
	Upload pictures.UploadID
	URL    string
}

func New(ids pictures.IDProvider, rules pictures.Rules, bucket Bucket) *UseCase {
	return &UseCase{ids: ids, rules: rules, bucket: bucket}
}

type UseCase struct {
	ids    pictures.IDProvider
	rules  pictures.Rules
	bucket Bucket
}

func (u *UseCase) Execute(in In) (Out, error) {
	if err := u.rules.Check(in.ContentType, in.Size); err != nil {
		return Out{}, err //nolint:wrapcheck // the rules name it.
	}
	id, err := u.ids.NewID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw an upload id: %w", err)
	}
	url, err := u.bucket.UploadURL(pictures.File{ID: id, ContentType: in.ContentType, Size: in.Size})
	if err != nil {
		return Out{}, fmt.Errorf("failed to sign the upload: %w", err)
	}
	return Out{Upload: id, URL: url}, nil
}
