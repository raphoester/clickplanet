package dressing_wear_title_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/wear_title_usecase/dressing_wear_title"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada     = players.AccountID{15: 1}
	catalog = titles.NewCatalog()
)

func standing(id titles.ID) titles.Standing {
	standing, _ := catalog.StandingOf(id)
	return standing
}

func setup(t *testing.T, author players.Author) (*dressing_wear_title.Dressing, *inmemory_visit_storage.Storage) {
	t.Helper()

	clock := cptime.NewFixedClock(now)
	held := inmemory_title_store.New()
	require.NoError(t, held.Grant(t.Context(), titles.Holdings{ada: {"og", "settler"}}, now))
	wardrobe := wearing.NewWardrobe(inmemory_worn_title_store.New(), titles.NewBook(held, catalog), catalog)

	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.NewVisit(ada, wearing.AuthorOf(author, standing("settler")), "aaaaaa", "fr", now))

	return dressing_wear_title.New(wear_title_usecase.New(wardrobe, clock), visits), visits
}

func TestAWornTitleShowsOnTheRosterAtOnce(t *testing.T) {
	dressing, visits := setup(t, players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada", time.Time{}, false, 0), players.Streak{}))

	worn, err := dressing.Execute(t.Context(), ada, "og")

	require.NoError(t, err)
	assert.Equal(t, standing("og"), worn)
	require.Len(t, visits.Visits(), 1)
	assert.Equal(t, standing("og"), visits.Visits()[0].Author().Worn())
}

func TestARefusedTitleChangesNothingOnTheRoster(t *testing.T) {
	dressing, visits := setup(t, players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada", time.Time{}, false, 0), players.Streak{}))

	_, err := dressing.Execute(t.Context(), ada, "warmaster")

	require.ErrorIs(t, err, wearing.ErrNotWearable)
	require.Len(t, visits.Visits(), 1)
	assert.Equal(t, standing("settler"), visits.Visits()[0].Author().Worn())
}

func TestAGuestLineWearsNothing(t *testing.T) {
	dressing, visits := setup(t, players.GuestAuthor("a1b2c3", players.Streak{}))

	_, err := dressing.Execute(t.Context(), ada, "og")

	require.NoError(t, err)
	require.Len(t, visits.Visits(), 1)
	assert.True(t, visits.Visits()[0].Author().Worn().Empty())
}
