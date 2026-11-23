package cpflags_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
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

func answersOf(t *testing.T, catalog cpflags.Catalog, flag string) answers {
	t.Helper()
	playable, err := catalog.Playable(t.Context(), flag)
	require.NoError(t, err)
	known, err := catalog.Known(t.Context(), flag)
	require.NoError(t, err)
	return answers{playable: playable, known: known}
}

func TestAFlagTheGameShipsWithIsPlayable(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	for _, flag := range []string{"fr", "xb", "gb-eng"} {
		assert.Equal(t, answers{playable: true, known: true}, answersOf(t, catalog, flag), flag)
	}
}

func TestAFlagNoFactNamedIsUnknown(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	assert.Equal(t, answers{}, answersOf(t, catalog, brittanyCity))
	assert.Equal(t, answers{}, answersOf(t, catalog, "zz"))
}

func TestAFlagHeardOpenedIsPlayable(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened))

	assert.Equal(t, answers{playable: true, known: true}, answersOf(t, catalog, brittanyCity))
}

func TestAFlagHeardClosedIsKnownAndNotPlayable(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, catalog.Close(t.Context(), brittanyCity, closed))

	assert.Equal(t, answers{known: true}, answersOf(t, catalog, brittanyCity))
}

func TestTheLatestFactHoldsWhateverOrderTheyAreHeardIn(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.NoError(t, catalog.Close(t.Context(), brittanyCity, closed))
	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened))

	assert.Equal(t, answers{known: true}, answersOf(t, catalog, brittanyCity),
		"each fact has a stream of its own: the opening may come after the closing")
}

func TestAFlagOpenedAgainAfterItsCloseIsPlayable(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, catalog.Close(t.Context(), brittanyCity, closed))
	require.NoError(t, catalog.Open(t.Context(), brittanyCity, closed.Add(time.Hour)))

	assert.Equal(t, answers{playable: true, known: true}, answersOf(t, catalog, brittanyCity))
}

func TestOnlyAFactThatChangesTheCatalogIsTold(t *testing.T) {
	told := &cpflags.Recorder{}
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), told)

	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened), "told again, at each boot")
	require.NoError(t, catalog.Close(t.Context(), brittanyCity, closed))
	require.NoError(t, catalog.Open(t.Context(), brittanyCity, opened), "older than the close")

	assert.Equal(t, 2, told.Told())
}

func TestAStoreThatCannotBeReadAnswersNothing(t *testing.T) {
	store := cpflags.NewMemoryStore()
	broken := errors.New("postgres is down")
	store.FailWith(broken)
	catalog := cpflags.NewCatalog(store, cpflags.Quiet{})

	_, err := catalog.Playable(t.Context(), brittanyCity)
	require.ErrorIs(t, err, broken)
	_, err = catalog.Known(t.Context(), brittanyCity)
	require.ErrorIs(t, err, broken)
	require.ErrorIs(t, catalog.Open(t.Context(), brittanyCity, opened), broken)

	playable, err := catalog.Playable(t.Context(), "fr")
	require.NoError(t, err, "a flag the game ships with is never read")
	assert.True(t, playable)
}
