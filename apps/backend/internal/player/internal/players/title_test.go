package players_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	first   = players.FakeTitle{Key: "first", Tiles: 1}
	third   = players.FakeTitle{Key: "third", Tiles: 3}
	catalog = players.Catalog{first, third}
)

func TestTheCatalogNamesTheTitlesStatsEarnInItsOrder(t *testing.T) {
	assert.Empty(t, catalog.EarnedBy(players.Career{}))
	assert.Equal(t, players.TitleIDs{"first"}, catalog.EarnedBy(tiles(2)))
	assert.Equal(t, players.TitleIDs{"third", "first"}, players.Catalog{third, first}.EarnedBy(tiles(3)))
}

func TestGrantsLeaveOutTheAccountsThatEarnNothing(t *testing.T) {
	grants := catalog.GrantsFor([]players.Career{
		{Stats: players.Stats{Account: players.AccountID{15: 1}, TilesTaken: 0}},
		{Stats: players.Stats{Account: players.AccountID{15: 2}, TilesTaken: 1}},
		{Stats: players.Stats{Account: players.AccountID{15: 3}, TilesTaken: 5}},
	})

	assert.Equal(t, players.Grants{
		{15: 2}: {"first"},
		{15: 3}: {"first", "third"},
	}, grants)
}

func TestTheHeldTitlesAreTheCatalogsInItsOrderAndAnUnknownOneIsDropped(t *testing.T) {
	held := catalog.Of(players.TitleIDs{"third", "retired", "first"})

	assert.Equal(t, []players.Title{first, third}, held)
}

func TestTheCatalogWithoutSomeTitlesKeepsTheRest(t *testing.T) {
	assert.Equal(t, players.Catalog{third}, catalog.Without(players.TitleIDs{"first"}))
	assert.Empty(t, catalog.Without(catalog.IDs()))
	assert.Equal(t, players.Catalog{first, third}, catalog, "the receiver is left as it was")
}

func TestWithoutLeavesOutTheHeldIDsAndKeepsTheOrder(t *testing.T) {
	earned := players.TitleIDs{"a", "b", "c"}

	assert.Equal(t, players.TitleIDs{"a", "c"}, earned.Without(players.TitleIDs{"b"}))
	assert.Empty(t, earned.Without(earned))
	assert.Equal(t, players.TitleIDs{"a", "b", "c"}, earned, "the receiver is left as it was")
}
