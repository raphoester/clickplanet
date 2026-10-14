package notes_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	rules = notes.AttachmentRules{MaxPerNote: 3, MaxBytes: 1000}
	photo = notes.Attachment{ID: notes.AttachmentID{15: 1}, MediaType: notes.MediaPNG, Size: 400}
	video = notes.Attachment{ID: notes.AttachmentID{15: 2}, MediaType: notes.MediaMP4, Size: 1000}
)

func sent(t *testing.T) notes.Note {
	t.Helper()

	note, err := valid().Restore()
	require.NoError(t, err)
	return note
}

func photoNumbered(n byte) notes.Attachment {
	file := photo
	file.ID = notes.AttachmentID{14: 1, 15: n}
	return file
}

func TestANoteCarriesTheFilesAttachedToItInTheirOrder(t *testing.T) {
	note := sent(t)

	carrying, err := note.Attaching([]notes.Attachment{video, photo}, rules)

	require.NoError(t, err)
	want := valid()
	want.Attachments = []notes.Attachment{video, photo}
	assert.Equal(t, want, carrying.TakeSnapshot())
	assert.Equal(t, valid(), note.TakeSnapshot(), "the note it was made from is left as it was")
}

func TestANoteWithNothingAttachedIsTheNoteItWas(t *testing.T) {
	note := sent(t)

	for _, none := range [][]notes.Attachment{nil, {}} {
		same, err := note.Attaching(none, notes.AttachmentRules{})

		require.NoError(t, err)
		assert.Equal(t, note, same)
	}
}

func TestTheRulesRefuseOneFileTooManyAndOneByteTooMany(t *testing.T) {
	big := notes.Attachment{ID: notes.AttachmentID{15: 9}, MediaType: notes.MediaMP4, Size: 1001}

	for name, refused := range map[string]struct {
		files []notes.Attachment
		rules notes.AttachmentRules
	}{
		"four files":                {files: []notes.Attachment{photoNumbered(1), photoNumbered(2), photoNumbered(3), photoNumbered(4)}, rules: rules},
		"a file too big":            {files: []notes.Attachment{photo, big}, rules: rules},
		"a file when none is taken": {files: []notes.Attachment{photo}, rules: notes.AttachmentRules{}},
	} {
		_, err := sent(t).Attaching(refused.files, refused.rules)

		require.ErrorIs(t, err, notes.ErrInvalidAttachment, name)
	}

	_, err := sent(t).Attaching([]notes.Attachment{photo, video, photoNumbered(3)}, rules)
	require.NoError(t, err, "three files, one of them at the limit")
}

func TestAFileWithNoIDOfAnUnknownTypeOrOfNoSizeIsNoAttachment(t *testing.T) {
	for name, file := range map[string]notes.Attachment{
		"no id":                          {MediaType: notes.MediaPNG, Size: 10},
		"a drawing that can run scripts": {ID: notes.AttachmentID{15: 1}, MediaType: "image/svg+xml", Size: 10},
		"no type":                        {ID: notes.AttachmentID{15: 1}, Size: 10},
		"no size":                        {ID: notes.AttachmentID{15: 1}, MediaType: notes.MediaPNG},
		"a size below zero":              {ID: notes.AttachmentID{15: 1}, MediaType: notes.MediaPNG, Size: -1},
	} {
		_, err := sent(t).Attaching([]notes.Attachment{file}, rules)
		require.ErrorIs(t, err, notes.ErrInvalidAttachment, name)

		kept := valid()
		kept.Attachments = []notes.Attachment{file}
		_, err = kept.Restore()
		require.ErrorIs(t, err, notes.ErrInvalidAttachment, name)

		require.ErrorIs(t, rules.UploadError(file), notes.ErrInvalidAttachment, name)
	}
}

func TestOneFileIsAttachedOnceToANote(t *testing.T) {
	_, err := sent(t).Attaching([]notes.Attachment{photo, video, photo}, rules)

	require.ErrorIs(t, err, notes.ErrInvalidAttachment)
}

func TestANoteKeptWithMoreThanTheRulesNowTakeIsStillRestored(t *testing.T) {
	kept := valid()
	kept.Attachments = []notes.Attachment{
		{ID: notes.AttachmentID{15: 9}, MediaType: notes.MediaMP4, Size: 1 << 40},
		photoNumbered(1), photoNumbered(2), photoNumbered(3), photoNumbered(4),
	}

	note, err := kept.Restore()

	require.NoError(t, err)
	assert.Equal(t, kept, note.TakeSnapshot())
}

func TestAFileIsUploadedWhenTheRulesWouldTakeItInANote(t *testing.T) {
	require.NoError(t, rules.UploadError(photo))
	require.NoError(t, rules.UploadError(video), "a file at the limit")

	bigger := video
	bigger.Size++
	require.ErrorIs(t, rules.UploadError(bigger), notes.ErrInvalidAttachment)
	require.ErrorIs(t, notes.AttachmentRules{}.UploadError(photo), notes.ErrInvalidAttachment, "a server that takes no file")
}

func TestAnAttachmentIDIsReadFromItsText(t *testing.T) {
	id := uuid.New()

	read, err := notes.AttachmentIDOf(id.String())

	require.NoError(t, err)
	assert.Equal(t, notes.AttachmentID(id), read)
	assert.Equal(t, id.String(), read.String())

	for _, text := range []string{"", "a file", uuid.Nil.String()} {
		_, err := notes.AttachmentIDOf(text)
		require.ErrorIs(t, err, notes.ErrInvalidAttachment, text)
	}
}

func TestEveryMediaTypeIsAPhotoOrAVideoAndTheListIsACopy(t *testing.T) {
	types := notes.MediaTypes()
	require.NotEmpty(t, types)
	for _, media := range types {
		assert.Regexp(t, `^(image|video)/[a-z0-9]+$`, string(media))
	}

	types[0] = "text/html"
	assert.NotEqual(t, notes.MediaType("text/html"), notes.MediaTypes()[0])
}

func TestTheSnapshotOfANoteDoesNotShareItsFiles(t *testing.T) {
	files := []notes.Attachment{photo}
	note, err := sent(t).Attaching(files, rules)
	require.NoError(t, err)

	files[0] = video
	note.TakeSnapshot().Attachments[0] = video

	assert.Equal(t, []notes.Attachment{photo}, note.TakeSnapshot().Attachments)
}

func TestThePageIsToldNoFileBiggerThanTheRoomLeftAndNoneOnceItIsFull(t *testing.T) {
	rules := notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 1000}

	assert.Equal(t, rules, rules.Within(5000))
	assert.Equal(t, notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 300}, rules.Within(300))
	assert.Equal(t, notes.AttachmentRules{}, rules.Within(0))
	assert.Equal(t, notes.AttachmentRules{}, notes.AttachmentRules{}.Within(5000), "a server that takes no file")
}
