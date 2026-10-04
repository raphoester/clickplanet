package wear_title_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func useCaseHolding(t *testing.T, held ...titles.ID) *wear_title_usecase.UseCase {
	t.Helper()

	owned := inmemory_title_store.New()
	require.NoError(t, owned.Grant(t.Context(), titles.Holdings{ada: held}, now))
	catalog := titles.NewCatalog()
	wardrobe := wearing.NewWardrobe(inmemory_worn_title_store.New(), titles.NewBook(owned, catalog), catalog)
	return wear_title_usecase.New(wardrobe, cptime.NewFixedClock(now))
}

func TestWearingATitleAnswersTheTitleNowWorn(t *testing.T) {
	worn, err := useCaseHolding(t, "og", "settler", "raider").Execute(t.Context(), ada, "og")

	require.NoError(t, err)
	og, _ := titles.NewCatalog().StandingOf("og")
	assert.Equal(t, og, worn)
}

func TestATitleThatCannotBeWornIsRefused(t *testing.T) {
	_, err := useCaseHolding(t, "settler", "raider").Execute(t.Context(), ada, "settler")

	assert.ErrorIs(t, err, wearing.ErrNotWearable, "a rank below the one shown")
}
