package attachment_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/attachment"
)

var (
	first  = uuid.MustParse("6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a01")
	second = uuid.MustParse("6f1c2c64-6a4a-4b0e-9f43-2f0d7c1b5a02")
)

func TestEachIDARequestNamesIsHandedOnInItsOrder(t *testing.T) {
	ids, err := attachment.IDs([]string{second.String(), first.String()})

	require.NoError(t, err)
	assert.Equal(t, []notes.AttachmentID{notes.AttachmentID(second), notes.AttachmentID(first)}, ids)
}

func TestNoIDIsNoAttachment(t *testing.T) {
	ids, err := attachment.IDs(nil)

	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestAnIDThatIsNotOneIsRefused(t *testing.T) {
	for _, sent := range []string{"", "a file", uuid.Nil.String()} {
		_, err := attachment.IDs([]string{first.String(), sent})

		require.ErrorIs(t, err, notes.ErrInvalidAttachment, sent)
	}
}

func TestAnUploadIsAnsweredWithItsIDAndItsLink(t *testing.T) {
	answer := attachment.Uploaded(notes.AttachmentID(first), "https://bucket.example/6f1c")

	assert.Equal(t, first.String(), answer.GetAttachmentId())
	assert.Equal(t, "https://bucket.example/6f1c", answer.GetUrl())
}
