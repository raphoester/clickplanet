package catalog_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
)

const brittanyCity = "r-k3p9x2ma0b"

var (
	opened = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closed = opened.Add(time.Hour)
)

type answers struct {
	playable bool
	known    bool
}

func answersOf(t *testing.T, flags catalog.Catalog, flag string) answers {
	t.Helper()
	playable, err := flags.Playable(t.Context(), flag)
	require.NoError(t, err)
	known, err := flags.Known(t.Context(), flag)
	require.NoError(t, err)
	return answers{playable: playable, known: known}
}

func TestAFlagTheGameShipsWithIsPlayable(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	for _, flag := range []string{"fr", "xb", "gb-eng"} {
		assert.Equal(t, answers{playable: true, known: true}, answersOf(t, flags, flag), flag)
	}
}

func TestAFlagNoFactNamedIsUnknown(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	assert.Equal(t, answers{}, answersOf(t, flags, brittanyCity))
	assert.Equal(t, answers{}, answersOf(t, flags, "zz"))
}

func TestAFlagHeardOpenedIsPlayable(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	require.NoError(t, flags.Open(t.Context(), brittanyCity, opened))

	assert.Equal(t, answers{playable: true, known: true}, answersOf(t, flags, brittanyCity))
}

func TestAFlagHeardClosedIsKnownAndNotPlayable(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	require.NoError(t, flags.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, flags.Close(t.Context(), brittanyCity, closed))

	assert.Equal(t, answers{known: true}, answersOf(t, flags, brittanyCity))
}

func TestTheLatestFactHoldsWhateverOrderTheyAreHeardIn(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	require.NoError(t, flags.Close(t.Context(), brittanyCity, closed))
	require.NoError(t, flags.Open(t.Context(), brittanyCity, opened))

	assert.Equal(t, answers{known: true}, answersOf(t, flags, brittanyCity),
		"each fact has a stream of its own: the opening may come after the closing")
}

func TestAFlagOpenedAgainAfterItsCloseIsPlayable(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	require.NoError(t, flags.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, flags.Close(t.Context(), brittanyCity, closed))
	require.NoError(t, flags.Open(t.Context(), brittanyCity, closed.Add(time.Hour)))

	assert.Equal(t, answers{playable: true, known: true}, answersOf(t, flags, brittanyCity))
}

func TestAStoreThatCannotBeReadAnswersNothing(t *testing.T) {
	store := catalog.NewMemoryStore()
	broken := errors.New("postgres is down")
	store.FailWith(broken)
	flags := catalog.NewCatalog(store, catalog.Unlocked)

	_, err := flags.Playable(t.Context(), brittanyCity)
	require.ErrorIs(t, err, broken)
	_, err = flags.Known(t.Context(), brittanyCity)
	require.ErrorIs(t, err, broken)
	require.ErrorIs(t, flags.Open(t.Context(), brittanyCity, opened), broken)

	playable, err := flags.Playable(t.Context(), "fr")
	require.NoError(t, err, "a flag the game ships with is never read")
	assert.True(t, playable)
}
