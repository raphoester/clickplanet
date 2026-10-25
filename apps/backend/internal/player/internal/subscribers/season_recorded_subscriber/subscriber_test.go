package season_recorded_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/season_recorded_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 31, 23, 1, 0, 0, time.UTC)

func TestARecordedSeasonIsCheckedForTitles(t *testing.T) {
	stats, records, held, accounts := inmemory_player_store.New(), inmemory_season_store.New(), inmemory_title_store.New(), titles.NewFakeAccounts()
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	accounts.Create(account, players.AccountOf(true, now))
	require.NoError(t, stats.Record(t.Context(), account, players.TakeAt(now.Add(-time.Hour))))
	require.NoError(t, records.Save(t.Context(), seasons.NewRecord(account).With(seasons.Season{Number: 0, Flag: "fr"})))
	award := award_titles_usecase.New(stats, titles.NewCareers(accounts, records),
		titles.NewBook(held, titles.CatalogOf([]titles.Title{titles.Veteran{Season: 0}})), cptime.NewFixedClock(now))

	err = season_recorded_subscriber.New(award).Handle(t.Context(), &playerv1.SeasonRecorded{AccountId: account.String()})

	require.NoError(t, err)
	granted, err := held.Held(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"veteran_s0"}, granted)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	award := award_titles_usecase.New(inmemory_player_store.New(), titles.NewCareers(titles.NewFakeAccounts(), inmemory_season_store.New()),
		titles.NewBook(inmemory_title_store.New(), titles.NewCatalog()), cptime.NewFixedClock(now))

	err := season_recorded_subscriber.New(award).Handle(t.Context(), &playerv1.SeasonRecorded{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
