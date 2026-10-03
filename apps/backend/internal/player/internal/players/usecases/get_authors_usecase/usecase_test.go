package get_authors_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_authors_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ada     = players.AccountID{15: 1}
	bob     = players.AccountID{15: 2}
	guest   = players.AccountID{15: 3}
	today   = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	catalog = titles.NewCatalog()
)

type fixture struct {
	players *inmemory_player_store.Store
	held    *inmemory_title_store.Store
	worn    *inmemory_worn_title_store.Store
	useCase *get_authors_usecase.UseCase
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	f := fixture{players: inmemory_player_store.New(), held: inmemory_title_store.New(), worn: inmemory_worn_title_store.New()}
	require.NoError(t, f.players.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada", UpdatedAt: today}))
	require.NoError(t, f.players.SaveProfile(t.Context(), players.Profile{Account: bob, Name: "Bob", UpdatedAt: today}))
	require.NoError(t, f.players.SaveGuestCode(t.Context(), guest, "a1b2c3"))
	require.NoError(t, f.held.Grant(t.Context(), titles.Holdings{ada: {"og", "settler"}, guest: {"og"}}, today))
	require.NoError(t, f.worn.Wear(t.Context(), ada, "og", today))

	wardrobe := wearing.NewWardrobe(f.worn, titles.NewBook(f.held, catalog), catalog)
	f.useCase = get_authors_usecase.New(f.players, wardrobe, cptime.NewFixedClock(today))
	return f
}

func standing(id titles.ID) titles.Standing {
	standing, _ := catalog.StandingOf(id)
	return standing
}

func TestEachAuthorWearsItsTitleAndAGuestWearsNone(t *testing.T) {
	f := newFixture(t)

	authors, err := f.useCase.Execute(t.Context(), []players.AccountID{ada, bob, guest})

	require.NoError(t, err)
	assert.Equal(t, map[players.AccountID]wearing.Author{
		ada:   {Author: players.Author{Name: "Ada"}, Worn: standing("og")},
		bob:   {Author: players.Author{Name: "Bob"}},
		guest: {Author: players.Author{Name: "guest_a1b2c3", Guest: true}},
	}, authors)
}

func TestATitleStoreFailureIsAnError(t *testing.T) {
	f := newFixture(t)
	f.worn.FailWith(errors.New("postgres is down"))

	_, err := f.useCase.Execute(t.Context(), []players.AccountID{ada})

	assert.Error(t, err)
}
