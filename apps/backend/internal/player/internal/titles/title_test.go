package titles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

var (
	first   = titles.FakeTitle{Key: "first", Tiles: 1}
	third   = titles.FakeTitle{Key: "third", Tiles: 3}
	catalog = titles.Catalog{first, third}
)

func TestTheCatalogNamesTheTitlesStatsEarnInItsOrder(t *testing.T) {
	assert.Empty(t, catalog.EarnedBy(titles.Career{}))
	assert.Equal(t, titles.IDs{"first"}, catalog.EarnedBy(tiles(2)))
	assert.Equal(t, titles.IDs{"third", "first"}, titles.Catalog{third, first}.EarnedBy(tiles(3)))
}

func TestAGuestEarnsNothing(t *testing.T) {
	guest := titles.Career{Stats: players.Stats{TilesTaken: 1_000_000, StreakBest: 1_000}}

	assert.Empty(t, catalog.EarnedBy(guest))
	assert.Empty(t, titles.NewCatalog().EarnedBy(guest))
}

func TestTheReconciliationGrantsWhatIsEarnedAndRevokesWhatIsNot(t *testing.T) {
	career := func(account byte, taken uint64, linked bool) titles.Career {
		return titles.Career{
			Stats:   players.Stats{Account: players.AccountID{15: account}, TilesTaken: taken},
			Account: players.Account{Linked: linked},
		}
	}

	reconciliation := catalog.ReconciliationOf([]titles.Career{
		career(1, 5, true),
		career(2, 1, true),
		career(3, 5, false),
		career(4, 0, true),
	}, titles.Holdings{
		{15: 1}: {"first"},
		{15: 2}: {"first", "third", "retired"},
		{15: 3}: {"first"},
	})

	assert.Equal(t, titles.Holdings{{15: 1}: {"third"}}, reconciliation.Grants)
	assert.Equal(t, titles.Holdings{{15: 2}: {"third", "retired"}, {15: 3}: {"first"}}, reconciliation.Revocations)
	assert.Equal(t, 3, reconciliation.Revocations.Len())
}

func TestTheHeldTitlesAreTheCatalogsInItsOrderAndAnUnknownOneIsDropped(t *testing.T) {
	held := catalog.Of(titles.IDs{"third", "retired", "first"})

	assert.Equal(t, []titles.Title{first, third}, held)
}

func TestWithoutLeavesOutTheHeldIDsAndKeepsTheOrder(t *testing.T) {
	earned := titles.IDs{"a", "b", "c"}

	assert.Equal(t, titles.IDs{"a", "c"}, earned.Without(titles.IDs{"b"}))
	assert.Empty(t, earned.Without(earned))
	assert.Equal(t, titles.IDs{"a", "b", "c"}, earned, "the receiver is left as it was")
}
