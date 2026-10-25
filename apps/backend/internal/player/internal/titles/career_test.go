package titles_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

func TestEachCareerIsItsStatsAndWhatAuthSaysOfItsAccount(t *testing.T) {
	known := players.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	page := []players.Stats{
		players.StatsSnapshot{Account: players.AccountID{15: 1}, TilesTaken: 3}.Restore(),
		players.NewStats(players.AccountID{15: 2}),
	}

	careers := titles.CareersOf(page, map[players.AccountID]players.Account{{15: 1}: known})

	assert.Equal(t, []titles.Career{titles.CareerOf(page[0], known), titles.CareerOf(page[1], players.Account{})}, careers,
		"an account auth does not know is a guest")
	assert.Equal(t, []players.AccountID{{15: 1}, {15: 2}}, players.AccountsOf(page))
}
