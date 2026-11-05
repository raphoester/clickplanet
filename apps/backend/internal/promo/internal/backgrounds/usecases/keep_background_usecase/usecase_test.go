package keep_background_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/keep_background_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at    = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	rules = backgrounds.Rules{MaxBytes: 2 << 20}
	war   = backgrounds.File{ID: backgrounds.BackgroundID{15: 1}, MediaType: "image/webp", Size: 300_000}
)

func TestABackgroundIsKeptAsTheBucketHoldsIt(t *testing.T) {
	bucket, store := backgrounds.NewMemoryBucket(), backgrounds.NewMemoryStore()
	bucket.Put(war)

	err := keep_background_usecase.New(bucket, store, rules, cptime.NewFixedClock(at)).Execute(t.Context(),
		keep_background_usecase.In{ID: war.ID, Name: "The war"})

	require.NoError(t, err)
	kept, ok := store.Kept(war.ID)
	require.True(t, ok)
	assert.Equal(t, backgrounds.BackgroundSnapshot{File: war, Name: "The war", KeptAt: at}, kept)
}

func TestAFileNeverSentIsNotKept(t *testing.T) {
	store := backgrounds.NewMemoryStore()

	err := keep_background_usecase.New(backgrounds.NewMemoryBucket(), store, rules, cptime.NewFixedClock(at)).Execute(
		t.Context(), keep_background_usecase.In{ID: war.ID, Name: "The war"})

	require.ErrorIs(t, err, backgrounds.ErrNotUploaded)
	_, ok := store.Kept(war.ID)
	assert.False(t, ok)
}

func TestTheRulesAreCheckedOnWhatTheBucketHolds(t *testing.T) {
	bucket, store := backgrounds.NewMemoryBucket(), backgrounds.NewMemoryStore()
	bucket.Put(backgrounds.File{ID: war.ID, MediaType: "image/gif", Size: war.Size})

	err := keep_background_usecase.New(bucket, store, rules, cptime.NewFixedClock(at)).Execute(t.Context(),
		keep_background_usecase.In{ID: war.ID, Name: "The war"})

	require.ErrorIs(t, err, backgrounds.ErrInvalidBackground)
}
