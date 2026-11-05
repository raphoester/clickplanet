package remove_background_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/remove_background_usecase"
)

var war = backgrounds.File{ID: backgrounds.BackgroundID{15: 1}, MediaType: "image/webp", Size: 300_000}

func kept(t *testing.T) (*backgrounds.MemoryBucket, *backgrounds.MemoryStore) {
	t.Helper()
	bucket, store := backgrounds.NewMemoryBucket(), backgrounds.NewMemoryStore()
	bucket.Put(war)
	background, err := backgrounds.Keep(war, "The war", time.Now(), backgrounds.Rules{MaxBytes: 2 << 20})
	require.NoError(t, err)
	require.NoError(t, store.Keep(t.Context(), background))
	return bucket, store
}

func TestABackgroundRemovedLeavesTheLibraryAndTheBucket(t *testing.T) {
	bucket, store := kept(t)

	require.NoError(t, remove_background_usecase.New(bucket, store).Execute(t.Context(), war.ID))

	_, ok := store.Kept(war.ID)
	assert.False(t, ok)
	assert.False(t, bucket.Holds(war.ID))
}

func TestABackgroundACardShowsStays(t *testing.T) {
	bucket, store := kept(t)
	store.Shown[war.ID] = []string{"urf"}

	err := remove_background_usecase.New(bucket, store).Execute(t.Context(), war.ID)

	require.ErrorIs(t, err, backgrounds.ErrShown)
	_, ok := store.Kept(war.ID)
	assert.True(t, ok)
	assert.True(t, bucket.Holds(war.ID), "its file stays too")
}

func TestABackgroundNotKeptIsNone(t *testing.T) {
	err := remove_background_usecase.New(backgrounds.NewMemoryBucket(), backgrounds.NewMemoryStore()).Execute(
		t.Context(), war.ID)

	require.ErrorIs(t, err, backgrounds.ErrNoBackground)
}
