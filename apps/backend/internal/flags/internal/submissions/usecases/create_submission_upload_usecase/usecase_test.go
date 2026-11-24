package create_submission_upload_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/create_submission_upload_usecase"
)

type ids struct{}

func (ids) NewID() (pictures.UploadID, error) { return pictures.UploadID{1}, nil }

func useCase() *create_submission_upload_usecase.UseCase {
	return create_submission_upload_usecase.New(
		create_upload_usecase.New(ids{}, pictures.Rules{MaxBytes: 5 << 20}, pictures.NewMemoryBucket()),
	)
}

func TestASignedInAccountGetsALinkToUploadItsFlag(t *testing.T) {
	out, err := useCase().Execute(create_submission_upload_usecase.In{
		Account: "00000000-0000-0000-0000-000000000001", Linked: true, ContentType: "image/png", Size: 1000,
	})
	require.NoError(t, err)
	assert.Equal(t, pictures.UploadID{1}, out.Upload)
	assert.Equal(t, "https://bucket.test/flags/uploads/01000000-0000-0000-0000-000000000000", out.URL)
}

func TestAGuestGetsNoLink(t *testing.T) {
	_, err := useCase().Execute(create_submission_upload_usecase.In{
		Account: "00000000-0000-0000-0000-000000000001", ContentType: "image/png", Size: 1000,
	})
	require.ErrorIs(t, err, submissions.ErrGuest)
}
