package create_upload_usecase

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

type Uploads interface {
	UploadURL(file videos.File) (string, error)
}

type In struct {
	MediaType videos.MediaType
	Size      int64
}

type Out struct {
	ID  videos.VideoID
	URL string
}

func New(ids videos.IDProvider, rules videos.Rules, uploads Uploads) *UseCase {
	return &UseCase{ids: ids, rules: rules, uploads: uploads}
}

type UseCase struct {
	ids     videos.IDProvider
	rules   videos.Rules
	uploads Uploads
}

func (u *UseCase) Execute(in In) (Out, error) {
	id, err := u.ids.NewID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw a video id: %w", err)
	}

	file := videos.File{ID: id, MediaType: in.MediaType, Size: in.Size}
	if err := u.rules.FileError(file); err != nil {
		return Out{}, fmt.Errorf("cannot take the video: %w", err)
	}

	link, err := u.uploads.UploadURL(file)
	if err != nil {
		return Out{}, fmt.Errorf("failed to make the upload link: %w", err)
	}
	return Out{ID: id, URL: link}, nil
}
