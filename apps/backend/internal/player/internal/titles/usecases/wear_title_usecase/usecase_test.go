package wear_title_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func TestWearingATitleAnswersTheTitleNowWorn(t *testing.T) {
	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: {"og", "settler", "raider"}}, now))

	worn, err := wear_title_usecase.New(titles.NewBook(store, titles.NewCatalog()), cptime.NewFixedClock(now)).Execute(t.Context(), ada, "og")

	require.NoError(t, err)
	assert.Equal(t, titles.Standing{Title: titles.OG{}}, worn)
}

func TestATitleThatCannotBeWornIsRefused(t *testing.T) {
	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: {"settler", "raider"}}, now))

	_, err := wear_title_usecase.New(titles.NewBook(store, titles.NewCatalog()), cptime.NewFixedClock(now)).Execute(t.Context(), ada, "settler")

	assert.ErrorIs(t, err, titles.ErrNotWearable, "a rank below the highest held")
}
