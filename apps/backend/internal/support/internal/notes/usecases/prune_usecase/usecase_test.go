package prune_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/prune_usecase"
)

var now = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

func suggestion(n int, at time.Time) notes.Snapshot {
	return notes.Snapshot{
		ID: notes.NoteID{14: byte(n >> 8), 15: byte(n)}, Kind: notes.KindSuggestion, Account: notes.AccountID{15: 1}, Text: "an idea", At: at,
	}
}

func TestAPruneForgetsTheNotesPastRetentionAndKeepsTheOthers(t *testing.T) {
	kept := inmemory_note_storage.New()
	fresh := suggestion(2, now.Add(-time.Hour))
	atTheLine := suggestion(3, now.Add(-24*time.Hour))
	kept.Seed(suggestion(1, now.Add(-48*time.Hour)), fresh, atTheLine)

	forgotten, err := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1), forgotten)
	assert.Equal(t, []notes.Snapshot{atTheLine, fresh}, kept.Kept())
}

func TestAPruneForgetsMoreNotesThanOneBatchHolds(t *testing.T) {
	kept := inmemory_note_storage.New()
	for n := range 1200 {
		kept.Seed(suggestion(n+1, now.Add(-48*time.Hour).Add(time.Duration(n)*time.Second)))
	}
	fresh := suggestion(2000, now.Add(-time.Hour))
	kept.Seed(fresh)

	forgotten, err := prune_usecase.New(24*time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, int64(1200), forgotten)
	assert.Equal(t, []notes.Snapshot{fresh}, kept.Kept())
}

func TestAFailedPruneSaysSo(t *testing.T) {
	kept := inmemory_note_storage.New()
	kept.FailWith(errors.New("postgres is down"))

	_, err := prune_usecase.New(time.Hour, cptime.NewFixedClock(now), kept).Execute(t.Context())

	require.Error(t, err)
}
