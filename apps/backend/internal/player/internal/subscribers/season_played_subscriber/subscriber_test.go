package season_played_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/usecases/record_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/season_played_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func subscriber(store *inmemory_season_store.Store) season_played_subscriber.Subscriber {
	return season_played_subscriber.New(record_season_usecase.New(store, cpbootstrap.NewRecordedEvents()))
}

func TestASeasonPlayedIsRecordedWithItsFlag(t *testing.T) {
	store := inmemory_season_store.New()

	err := subscriber(store).Handle(t.Context(), &seasonsv1.SeasonPlayed{Season: 0, AccountId: ada, MainFlag: "fr", Tiles: 12, Won: true})

	require.NoError(t, err)
	account, err := players.AccountIDOf(ada)
	require.NoError(t, err)
	record, err := store.Record(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, []seasons.Season{{Number: 0, Flag: "fr", Won: true}}, record.TakeSnapshot().Seasons)
}

func TestAnEventWithNoAccountOrNoFlagIsRefused(t *testing.T) {
	err := subscriber(inmemory_season_store.New()).Handle(t.Context(), &seasonsv1.SeasonPlayed{MainFlag: "fr"})
	assert.ErrorIs(t, err, players.ErrInvalidAccount)

	err = subscriber(inmemory_season_store.New()).Handle(t.Context(), &seasonsv1.SeasonPlayed{AccountId: ada})
	assert.ErrorContains(t, err, "the season has no flag")
}
