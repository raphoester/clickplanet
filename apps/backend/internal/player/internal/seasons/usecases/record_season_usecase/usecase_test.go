package record_season_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/usecases/record_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

var ada = players.AccountID{15: 1}

func TestEachSeasonIsRecordedOnceAndSaysSo(t *testing.T) {
	store := inmemory_season_store.New()
	events := cpbootstrap.NewRecordedEvents()
	useCase := record_season_usecase.New(store, events)
	played := record_season_usecase.In{Account: ada, Season: seasons.Season{Number: 0, Flag: "fr", Won: true}}

	require.NoError(t, useCase.Execute(t.Context(), played))
	require.NoError(t, useCase.Execute(t.Context(), played))

	record, err := store.Record(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, []seasons.Season{{Number: 0, Flag: "fr", Won: true}}, record.TakeSnapshot().Seasons, "a season told twice is kept once")
	require.Len(t, events.Published(), 2)
	assert.True(t, proto.Equal(&playerv1.SeasonRecorded{AccountId: ada.String()}, events.Published()[1]),
		"each telling says so, so the titles are checked again")
}

func TestAStoreFailureIsAnErrorAndSaysNothing(t *testing.T) {
	store := inmemory_season_store.New()
	store.FailWith(errors.New("postgres is down"))
	events := cpbootstrap.NewRecordedEvents()

	err := record_season_usecase.New(store, events).Execute(t.Context(), record_season_usecase.In{Account: ada})

	require.Error(t, err)
	assert.Empty(t, events.Published())
}
