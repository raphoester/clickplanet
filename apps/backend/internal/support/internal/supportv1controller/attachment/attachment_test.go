package attachment_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/attachment"
)

func TestEachFileARequestDeclaresIsHandedOnInItsOrder(t *testing.T) {
	declared := attachment.Declared([]*supportv1.Attachment{
		{ContentType: "video/mp4", SizeBytes: 9000}, {ContentType: "image/png", SizeBytes: 400}, {ContentType: "text/html"},
	})

	assert.Equal(t, []notes.Attachment{
		{MediaType: notes.MediaMP4, Size: 9000}, {MediaType: notes.MediaPNG, Size: 400}, {MediaType: "text/html"},
	}, declared)
	assert.Empty(t, attachment.Declared(nil))
}

func TestASizeNoFileHasStaysTooBigAndNeverTurnsNegative(t *testing.T) {
	declared := attachment.Declared([]*supportv1.Attachment{{ContentType: "video/mp4", SizeBytes: math.MaxUint64}})

	assert.Equal(t, int64(math.MaxInt64), declared[0].Size)
}

func TestEachLinkIsAnUploadInItsOrder(t *testing.T) {
	uploads := attachment.Uploads([]string{"https://bucket.example/a/0", "https://bucket.example/a/1"})

	assert.Len(t, uploads, 2)
	assert.Equal(t, "https://bucket.example/a/0", uploads[0].GetUrl())
	assert.Equal(t, "https://bucket.example/a/1", uploads[1].GetUrl())
	assert.Empty(t, attachment.Uploads(nil))
}
