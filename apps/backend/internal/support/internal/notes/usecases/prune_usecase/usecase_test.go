package prune_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase"
)

var now = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

func suggestion(n int, at time.Time) notes.Snapshot {
	return notes.Snapshot{
		ID: notes.NoteID{14: byte(n >> 8), 15: byte(n)}, Kind: notes.KindSuggestion, Account: notes.AccountID{15: 1}, Text: "an idea", At: at,
	}
}

func anonymous(note notes.Snapshot) notes.Snapshot {
	note.Account = notes.AccountID{}
	return note
}

func TestAPruneTakesTheSenderOffTheNotesPastRetentionAndLeavesTheOthersAlone(t *testing.T) {
	kept := inmemory_note_storage.New()
	old := suggestion(1, now.Add(-48*time.Hour))
	fresh := suggestion(2, now.Add(-time.Hour))
	atTheLine := suggestion(3, now.Add(-24*time.Hour))
	kept.Seed(old, fresh, atTheLine)

	anonymized, err := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept, inmemory_attachment_bucket.New()).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1), anonymized)
	assert.Equal(t, []notes.Snapshot{anonymous(old), atTheLine, fresh}, kept.Kept())
}

func TestASecondPruneFindsNothingLeftToDo(t *testing.T) {
	kept := inmemory_note_storage.New()
	kept.Seed(suggestion(1, now.Add(-48*time.Hour)))
	prune := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept, inmemory_attachment_bucket.New())
	_, err := prune.Execute(t.Context())
	require.NoError(t, err)

	anonymized, err := prune.Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, anonymized)
}

func TestAPruneGoesThroughMoreNotesThanOneBatchHolds(t *testing.T) {
	kept := inmemory_note_storage.New()
	for n := range 1200 {
		kept.Seed(suggestion(n+1, now.Add(-48*time.Hour).Add(time.Duration(n)*time.Second)))
	}
	fresh := suggestion(2000, now.Add(-time.Hour))
	kept.Seed(fresh)

	anonymized, err := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept, inmemory_attachment_bucket.New()).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1200), anonymized)
	after := kept.Kept()
	require.Len(t, after, 1201)
	assert.Equal(t, fresh, after[1200])
	for _, note := range after[:1200] {
		assert.Equal(t, notes.AccountID{}, note.Account)
	}
}

func TestAPruneDeletesTheFilesOfTheNotesPastRetentionAndNoOther(t *testing.T) {
	kept, bucket := inmemory_note_storage.New(), inmemory_attachment_bucket.New()
	old := suggestion(1, now.Add(-48*time.Hour))
	old.Attachments = []notes.Attachment{{MediaType: notes.MediaPNG, Size: 400}, {MediaType: notes.MediaMP4, Size: 9000}}
	fresh := suggestion(2, now.Add(-time.Hour))
	fresh.Attachments = []notes.Attachment{{MediaType: notes.MediaPNG, Size: 400}}
	kept.Seed(old, fresh)

	_, err := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept, bucket).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []string{
		inmemory_attachment_bucket.URL(old.ID, 0), inmemory_attachment_bucket.URL(old.ID, 1),
	}, bucket.Deleted())
	old.Attachments = nil
	assert.Equal(t, []notes.Snapshot{anonymous(old), fresh}, kept.Kept())
}

func TestAPruneThatCannotDeleteAFileLeavesTheNoteToTheNextOne(t *testing.T) {
	kept, bucket := inmemory_note_storage.New(), inmemory_attachment_bucket.New()
	old := suggestion(1, now.Add(-48*time.Hour))
	old.Attachments = []notes.Attachment{{MediaType: notes.MediaPNG, Size: 400}}
	kept.Seed(old)
	down := errors.New("the bucket is down")
	bucket.FailWith(down)
	prune := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept, bucket)

	anonymized, err := prune.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Zero(t, anonymized)
	assert.Equal(t, []notes.Snapshot{old}, kept.Kept())

	bucket.FailWith(nil)
	anonymized, err = prune.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), anonymized)
}

func TestAFailedPruneSaysSo(t *testing.T) {
	kept := inmemory_note_storage.New()
	kept.FailWith(errors.New("postgres is down"))

	_, err := prune_usecase.New(time.Hour, cptime.NewFixedClock(now), kept, inmemory_attachment_bucket.New()).Execute(t.Context())

	require.Error(t, err)
}
