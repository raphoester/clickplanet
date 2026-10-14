package create_upload_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/create_upload_usecase"
)

var rules = notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 1000}

type failingIDs struct{}

func (failingIDs) NewAttachmentID() (notes.AttachmentID, error) {
	return notes.AttachmentID{}, errors.New("no entropy")
}

func TestAFileTheRulesTakeGetsAFreshIDAndALinkToUploadItTo(t *testing.T) {
	ids := &notes.SequentialIDs{}
	useCase := create_upload_usecase.New(ids, rules, inmemory_attachment_bucket.New())

	first, err := useCase.Execute(create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 400})
	require.NoError(t, err)
	second, err := useCase.Execute(create_upload_usecase.In{MediaType: notes.MediaMP4, Size: 1000})
	require.NoError(t, err)

	assert.Equal(t, create_upload_usecase.Out{
		ID: notes.AttachmentID{0: 0xa, 15: 1}, URL: inmemory_attachment_bucket.URL(notes.AttachmentID{0: 0xa, 15: 1}),
	}, first)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestAFileTheRulesDoNotTakeGetsNoLink(t *testing.T) {
	for name, in := range map[string]create_upload_usecase.In{
		"too big":              {MediaType: notes.MediaMP4, Size: 1001},
		"not a photo or video": {MediaType: "text/html", Size: 10},
		"empty":                {MediaType: notes.MediaPNG},
	} {
		_, err := create_upload_usecase.New(&notes.SequentialIDs{}, rules, inmemory_attachment_bucket.New()).Execute(in)

		require.ErrorIs(t, err, notes.ErrInvalidAttachment, name)
	}

	_, err := create_upload_usecase.New(&notes.SequentialIDs{}, notes.AttachmentRules{}, inmemory_attachment_bucket.New()).
		Execute(create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.ErrorIs(t, err, notes.ErrInvalidAttachment, "a server that takes no file")
}

func TestAnUploadWithNoIDOrNoLinkFails(t *testing.T) {
	_, err := create_upload_usecase.New(failingIDs{}, rules, inmemory_attachment_bucket.New()).
		Execute(create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.Error(t, err)

	down := errors.New("no bucket")
	bucket := inmemory_attachment_bucket.New()
	bucket.FailWith(down)
	_, err = create_upload_usecase.New(&notes.SequentialIDs{}, rules, bucket).
		Execute(create_upload_usecase.In{MediaType: notes.MediaPNG, Size: 10})
	require.ErrorIs(t, err, down)
}
