package titles_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/inmemory_podium_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

func TestEachCareerIsItsStatsWhatAuthSaysOfItsAccountTheFinalesItFoughtAndItsPodiums(t *testing.T) {
	known := players.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	page := []players.Stats{
		players.StatsSnapshot{Account: players.AccountID{15: 1}, TilesTaken: 3}.Restore(),
		players.NewStats(players.AccountID{15: 2}),
	}
	fight, err := finales.FightSnapshot{Account: players.AccountID{15: 1}, Season: 0, Country: "fr", Rank: 1}.Restore()
	require.NoError(t, err)
	place, err := podiums.PlaceSnapshot{Account: players.AccountID{15: 1}, Season: 0, Rank: 2}.Restore()
	require.NoError(t, err)

	careers := titles.CareersOf(
		page,
		map[players.AccountID]players.Account{{15: 1}: known},
		map[players.AccountID]finales.Fights{{15: 1}: {fight}},
		map[players.AccountID]podiums.Places{{15: 1}: {place}},
	)

	assert.Equal(t, []titles.Career{
		titles.CareerOf(page[0], known, finales.Fights{fight}, podiums.Places{place}),
		titles.CareerOf(page[1], players.Account{}, nil, nil),
	}, careers, "an account auth does not know is a guest")
	assert.Equal(t, []players.AccountID{{15: 1}, {15: 2}}, players.AccountsOf(page))
}

func TestCareersReadTheAccountItsFightsAndItsPodiums(t *testing.T) {
	ada := players.AccountID{15: 1}
	known := players.AccountOf(true, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))
	accounts, fights, places := titles.NewFakeAccounts(), inmemory_fight_store.New(), inmemory_podium_store.New()
	accounts.Create(ada, known)
	fight, err := finales.FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 2}.Restore()
	require.NoError(t, err)
	require.NoError(t, fights.Record(t.Context(), []finales.Fight{fight}))
	place, err := podiums.PlaceSnapshot{Account: ada, Season: 0, Rank: 1}.Restore()
	require.NoError(t, err)
	require.NoError(t, places.Record(t.Context(), []podiums.Place{place}))
	stats := players.StatsSnapshot{Account: ada, TilesTaken: 3}.Restore()
	careers := titles.NewCareers(accounts, fights, places)

	career, err := careers.Of(t.Context(), stats)
	require.NoError(t, err)
	assert.Equal(t, titles.CareerOf(stats, known, finales.Fights{fight}, podiums.Places{place}), career)

	page, err := careers.PageOf(t.Context(), []players.Stats{stats})
	require.NoError(t, err)
	assert.Equal(t, []titles.Career{titles.CareerOf(stats, known, finales.Fights{fight}, podiums.Places{place})}, page)
}

func TestCareersFailWhenAuthTheFightsOrThePodiumsFail(t *testing.T) {
	stats := players.StatsSnapshot{Account: players.AccountID{15: 1}}.Restore()
	accounts, fights, places := titles.NewFakeAccounts(), inmemory_fight_store.New(), inmemory_podium_store.New()
	careers := titles.NewCareers(accounts, fights, places)

	fights.FailWith(errors.New("postgres is down"))
	_, err := careers.Of(t.Context(), stats)
	require.Error(t, err)
	_, err = careers.PageOf(t.Context(), []players.Stats{stats})
	require.Error(t, err)

	fights.FailWith(nil)
	places.FailWith(errors.New("postgres is down"))
	_, err = careers.Of(t.Context(), stats)
	require.Error(t, err)
	_, err = careers.PageOf(t.Context(), []players.Stats{stats})
	require.Error(t, err)

	places.FailWith(nil)
	accounts.FailWith(errors.New("auth is down"))
	_, err = careers.Of(t.Context(), stats)
	require.Error(t, err)
	_, err = careers.PageOf(t.Context(), []players.Stats{stats})
	assert.Error(t, err)
}
