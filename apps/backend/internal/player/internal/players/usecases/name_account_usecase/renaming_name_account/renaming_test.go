package renaming_name_account_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase/renaming_name_account"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ada   = players.AccountID{15: 1}
	guest = wearing.Author{Author: players.Author{Name: "guest_0b1c2d", Guest: true}}
)

func setup() (*inmemory_player_store.Store, *inmemory_visit_storage.Storage, *renaming_name_account.Renaming) {
	clock := cptime.NewFixedClock(now)
	store := inmemory_player_store.New()
	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.Visit{Account: ada, Author: guest, Tag: "aaaaaa", Country: "fr", At: now})
	names := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42"))
	return store, visits, renaming_name_account.New(name_account_usecase.New(names, store, clock), visits)
}

func TestAGeneratedNameShowsOnTheRosterAtOnce(t *testing.T) {
	_, visits, useCase := setup()

	_, err := useCase.Execute(t.Context(), ada)

	require.NoError(t, err)
	require.Len(t, visits.Visits(), 1)
	assert.Equal(t, wearing.Author{Author: players.Author{Name: "BraveFox42"}}, visits.Visits()[0].Author)
}

func TestAFailureRenamesNothing(t *testing.T) {
	store, visits, useCase := setup()
	store.FailWith(errors.New("postgres is down"))

	_, err := useCase.Execute(t.Context(), ada)

	require.Error(t, err)
	assert.Equal(t, guest, visits.Visits()[0].Author)
}
