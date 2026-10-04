package wearing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
)

var (
	at  = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}

	badge   = titles.FakeTitle{Key: "badge", Tiles: 0}
	low     = titles.FakeTitle{Key: "low", Tiles: 2}
	mid     = titles.FakeTitle{Key: "mid", Tiles: 5}
	high    = titles.FakeTitle{Key: "high", Tiles: 9}
	other   = titles.FakeTitle{Key: "other", Tiles: 4}
	ladder  = titles.FakeTrack{Key: "ladder", Steps: []titles.Rank{low, mid, high}}
	side    = titles.FakeTrack{Key: "side", Steps: []titles.Rank{other}}
	catalog = titles.CatalogOf([]titles.Title{badge}, ladder, side)
)

func standing(id titles.ID) titles.Standing {
	standing, _ := catalog.StandingOf(id)
	return standing
}

func TestTheWornTitleIsTheChoiceOrTheRankShownOnItsTrack(t *testing.T) {
	shown := catalog.Shown(titles.IDs{"badge", "low", "mid", "other"})

	assert.Equal(t, standing("badge"), wearing.WornOf(shown, standing("badge")))
	assert.Equal(t, standing("other"), wearing.WornOf(shown, standing("other")))
	assert.Equal(t, standing("mid"), wearing.WornOf(shown, standing("low")), "a track is worn at the rank it shows")
}

func TestWithNoChoiceShownTheFirstTrackShownIsWorn(t *testing.T) {
	assert.Equal(t, standing("mid"), wearing.WornOf(catalog.Shown(titles.IDs{"badge", "mid"}), titles.Standing{}))
	assert.Equal(t, standing("mid"), wearing.WornOf(catalog.Shown(titles.IDs{"badge", "mid"}), standing("other")),
		"a choice on a track the account no longer shows")
	assert.Equal(t, standing("badge"), wearing.WornOf(catalog.Shown(titles.IDs{"badge"}), titles.Standing{}), "then a standalone title")
	assert.True(t, wearing.WornOf(nil, standing("badge")).Empty(), "and nothing when nothing is shown")
}

func TestOnlyAShownTitleIsWearable(t *testing.T) {
	shown := catalog.Shown(titles.IDs{"badge", "low", "mid"})

	assert.True(t, wearing.Wearable(shown, "badge"))
	assert.True(t, wearing.Wearable(shown, "mid"))
	assert.False(t, wearing.Wearable(shown, "low"), "a rank below the one shown")
	assert.False(t, wearing.Wearable(shown, "high"), "a rank not held")
	assert.False(t, wearing.Wearable(shown, "retired"))
}

func wardrobe(t *testing.T, held ...titles.ID) (wearing.Wardrobe, *inmemory_title_store.Store, *inmemory_worn_title_store.Store) {
	t.Helper()

	owned := inmemory_title_store.New()
	require.NoError(t, owned.Grant(t.Context(), titles.Holdings{ada: held}, at))
	worn := inmemory_worn_title_store.New()
	return wearing.NewWardrobe(worn, titles.NewBook(owned, catalog), catalog), owned, worn
}

func TestTheShowcaseIsTheWornTitleAndWhatIsShown(t *testing.T) {
	closet, _, _ := wardrobe(t, "badge", "low", "mid")

	showcase, err := closet.Showcase(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, standing("mid"), showcase.Worn())
	assert.Equal(t, []titles.Standing{standing("badge"), standing("mid")}, showcase.Shown())
}

func TestWearingAShownTitleChangesTheShowcase(t *testing.T) {
	closet, _, _ := wardrobe(t, "badge", "low", "mid")

	require.NoError(t, closet.Wear(t.Context(), ada, "badge", at))

	showcase, err := closet.Showcase(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, standing("badge"), showcase.Worn())
}

func TestATitleNotShownCannotBeWornAndNothingIsWritten(t *testing.T) {
	closet, _, worn := wardrobe(t, "badge", "low", "mid")

	for _, id := range []titles.ID{"low", "high", "retired"} {
		assert.ErrorIs(t, closet.Wear(t.Context(), ada, id, at), wearing.ErrNotWearable, id)
	}
	choice, err := worn.Choice(t.Context(), ada)
	require.NoError(t, err)
	assert.Empty(t, choice)
}

func TestAShowcaseIsWhatTheCatalogShowsOfTheTitlesHeldAndTheOneWorn(t *testing.T) {
	held := titles.IDs{"badge", "low", "mid", "retired"}

	chosen := wearing.ShowcaseOf(catalog, held, "badge")
	assert.Equal(t, standing("badge"), chosen.Worn())
	assert.Equal(t, catalog.Shown(held), chosen.Shown())
	assert.Equal(t, standing("mid"), wearing.ShowcaseOf(catalog, held, "").Worn(), "no choice wears the first rank shown")
	assert.True(t, wearing.ShowcaseOf(catalog, nil, "badge").Worn().Empty(), "a choice not held is not worn")
}

func TestAnAuthorWearsItsTitleUnlessItIsAGuest(t *testing.T) {
	named := players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada", time.Time{}, false, 0), players.Streak{})
	guest := players.GuestAuthor("a1b2c3", players.Streak{})

	dressed := wearing.AuthorOf(named, standing("mid"))
	assert.Equal(t, standing("mid"), dressed.Worn())
	assert.Equal(t, "Ada", dressed.Name())
	assert.True(t, wearing.AuthorOf(guest, standing("mid")).Worn().Empty())
	assert.True(t, wearing.AuthorOf(guest, titles.Standing{}).Wearing(standing("mid")).Worn().Empty(), "nor once dressed")
}

func TestAStoreFailureIsAnError(t *testing.T) {
	closet, owned, _ := wardrobe(t, "badge")
	owned.FailWith(errors.New("postgres is down"))

	_, err := closet.Showcase(t.Context(), ada)
	require.Error(t, err)
	require.Error(t, closet.Wear(t.Context(), ada, "badge", at))

	closet, _, worn := wardrobe(t, "badge")
	worn.FailWith(errors.New("postgres is down"))

	_, err = closet.Showcase(t.Context(), ada)
	require.Error(t, err)
	assert.Error(t, closet.Wear(t.Context(), ada, "badge", at))
}
