package titles_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

func TestEachCareerIsItsStatsWhatAuthSaysOfItsAccountAndTheFinalesItFought(t *testing.T) {
	known := accounts.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	page := []stats.Stats{
		stats.StatsSnapshot{Account: accounts.AccountID{15: 1}, TilesTaken: 3}.Restore(),
		stats.NewStats(accounts.AccountID{15: 2}),
	}
	fight, err := finales.FightSnapshot{Account: accounts.AccountID{15: 1}, Season: 0, Country: "fr", Rank: 1}.Restore()
	require.NoError(t, err)

	careers := titles.CareersOf(
		page,
		map[accounts.AccountID]accounts.Account{{15: 1}: known},
		map[accounts.AccountID]finales.Fights{{15: 1}: {fight}},
	)

	assert.Equal(t, []titles.Career{
		titles.CareerOf(page[0], known, finales.Fights{fight}), titles.CareerOf(page[1], accounts.Account{}, nil),
	}, careers, "an account player does not answer is a guest")
	assert.Equal(t, []accounts.AccountID{{15: 1}, {15: 2}}, stats.AccountsOf(page))
}

func TestCareersReadTheAccountAndItsFights(t *testing.T) {
	ada := accounts.AccountID{15: 1}
	known := accounts.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	asked, fights := titles.NewFakeAccounts(), inmemory_fight_store.New()
	asked.Create(ada, known)
	fight, err := finales.FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 2}.Restore()
	require.NoError(t, err)
	require.NoError(t, fights.Record(t.Context(), []finales.Fight{fight}))
	counted := stats.StatsSnapshot{Account: ada, TilesTaken: 3}.Restore()
	careers := titles.NewCareers(asked, fights)

	career, err := careers.Of(t.Context(), counted)
	require.NoError(t, err)
	assert.Equal(t, titles.CareerOf(counted, known, finales.Fights{fight}), career)

	page, err := careers.PageOf(t.Context(), []stats.Stats{counted})
	require.NoError(t, err)
	assert.Equal(t, []titles.Career{titles.CareerOf(counted, known, finales.Fights{fight})}, page)
}
