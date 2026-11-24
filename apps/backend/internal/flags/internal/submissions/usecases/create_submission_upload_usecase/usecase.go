package create_submission_upload_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Uploads interface {
	Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error)
}

type In struct {
	Account     string
	Linked      bool
	ContentType string
	Size        int64
}

func New(uploads Uploads) *UseCase {
	return &UseCase{uploads: uploads}
}

// UseCase signs an upload for an account that may found a flag.
type UseCase struct {
	uploads Uploads
}

func (u *UseCase) Execute(in In) (create_upload_usecase.Out, error) {
	if _, err := submissions.FounderOf(in.Account, in.Linked); err != nil {
		return create_upload_usecase.Out{}, err //nolint:wrapcheck // the rule names it.
	}
	return u.uploads.Execute(create_upload_usecase.In{ContentType: in.ContentType, Size: in.Size}) //nolint:wrapcheck // the use case names it.
}
