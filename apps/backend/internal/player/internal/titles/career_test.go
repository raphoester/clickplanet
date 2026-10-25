package titles_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

func TestEachCareerIsItsStatsWhatAuthSaysOfItsAccountAndTheSeasonsItPlayed(t *testing.T) {
	known := players.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	played := seasons.NewRecord(players.AccountID{15: 1}).With(seasons.Season{Number: 0, Flag: "fr"})
	page := []players.Stats{
		players.StatsSnapshot{Account: players.AccountID{15: 1}, TilesTaken: 3}.Restore(),
		players.NewStats(players.AccountID{15: 2}),
	}

	careers := titles.CareersOf(page, map[players.AccountID]players.Account{{15: 1}: known},
		map[players.AccountID]seasons.Record{{15: 1}: played})

	assert.Equal(t, []titles.Career{
		titles.CareerOf(page[0], known, played),
		titles.CareerOf(page[1], players.Account{}, seasons.Record{}),
	}, careers, "an account auth does not know is a guest, and one with no record played nothing")
	assert.Equal(t, []players.AccountID{{15: 1}, {15: 2}}, players.AccountsOf(page))
}

func TestCareersAskAuthAndReadTheSeasonsPlayed(t *testing.T) {
	ada := players.AccountID{15: 1}
	known := players.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	accounts, records := titles.NewFakeAccounts(), inmemory_season_store.New()
	accounts.Create(ada, known)
	played := seasons.NewRecord(ada).With(seasons.Season{Number: 0, Flag: "fr", Won: true})
	require.NoError(t, records.Save(t.Context(), played))
	stats := players.StatsSnapshot{Account: ada, TilesTaken: 3}.Restore()
	careers := titles.NewCareers(accounts, records)

	career, err := careers.Of(t.Context(), stats)
	require.NoError(t, err)
	assert.Equal(t, titles.CareerOf(stats, known, played), career)

	page, err := careers.Page(t.Context(), []players.Stats{stats})
	require.NoError(t, err)
	assert.Equal(t, []titles.Career{career}, page)

	records.FailWith(errors.New("postgres is down"))
	_, err = careers.Of(t.Context(), stats)
	require.Error(t, err)
	_, err = careers.Page(t.Context(), []players.Stats{stats})
	require.Error(t, err)
}
