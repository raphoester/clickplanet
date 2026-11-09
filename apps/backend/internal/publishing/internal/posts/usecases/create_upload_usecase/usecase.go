package create_upload_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
)

type Uploads interface {
	MediaUpload(ctx context.Context, media posts.Media) (posts.MediaUpload, error)
}

func New(uploads Uploads) *UseCase {
	return &UseCase{uploads: uploads}
}

// Checks the file, then asks the service where to upload it: no byte of it passes through the server.
type UseCase struct {
	uploads Uploads
}

func (u *UseCase) Execute(ctx context.Context, media posts.Media) (posts.MediaUpload, error) {
	if err := media.FileError(); err != nil {
		return posts.MediaUpload{}, err //nolint:wrapcheck // the domain already says every reason.
	}
	upload, err := u.uploads.MediaUpload(ctx, media)
	if err != nil {
		return posts.MediaUpload{}, fmt.Errorf("failed to ask the service where to upload %q: %w", media.FileName, err)
	}
	return upload, nil
}
