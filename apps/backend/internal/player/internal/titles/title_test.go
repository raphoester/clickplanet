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

func TestGrantsLeaveOutTheAccountsThatEarnNothing(t *testing.T) {
	grants := catalog.GrantsFor([]titles.Career{
		{Stats: players.Stats{Account: players.AccountID{15: 1}, TilesTaken: 0}},
		{Stats: players.Stats{Account: players.AccountID{15: 2}, TilesTaken: 1}},
		{Stats: players.Stats{Account: players.AccountID{15: 3}, TilesTaken: 5}},
	})

	assert.Equal(t, titles.Grants{
		{15: 2}: {"first"},
		{15: 3}: {"first", "third"},
	}, grants)
}

func TestTheHeldTitlesAreTheCatalogsInItsOrderAndAnUnknownOneIsDropped(t *testing.T) {
	held := catalog.Of(titles.IDs{"third", "retired", "first"})

	assert.Equal(t, []titles.Title{first, third}, held)
}

func TestTheCatalogWithoutSomeTitlesKeepsTheRest(t *testing.T) {
	assert.Equal(t, titles.Catalog{third}, catalog.Without(titles.IDs{"first"}))
	assert.Empty(t, catalog.Without(catalog.IDs()))
	assert.Equal(t, titles.Catalog{first, third}, catalog, "the receiver is left as it was")
}

func TestWithoutLeavesOutTheHeldIDsAndKeepsTheOrder(t *testing.T) {
	earned := titles.IDs{"a", "b", "c"}

	assert.Equal(t, titles.IDs{"a", "c"}, earned.Without(titles.IDs{"b"}))
	assert.Empty(t, earned.Without(earned))
	assert.Equal(t, titles.IDs{"a", "b", "c"}, earned, "the receiver is left as it was")
}
