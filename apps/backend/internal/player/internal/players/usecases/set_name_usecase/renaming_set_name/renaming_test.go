package renaming_set_name_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase/renaming_set_name"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}

	guest = players.Author{Name: "guest_0b1c2d", Guest: true}
)

type fixture struct {
	accounts *set_name_usecase.FakeAccounts
	visits   *inmemory_visit_storage.Storage
	useCase  *renaming_set_name.Renaming
}

func setup() fixture {
	clock := cptime.NewFixedClock(now)
	accounts := set_name_usecase.NewFakeAccounts()
	visits := inmemory_visit_storage.New(clock)
	visits.Record(presence.Visit{Account: ada, Author: guest, Tag: "aaaaaa", Country: "fr", At: now})

	return fixture{
		accounts: accounts,
		visits:   visits,
		useCase:  renaming_set_name.New(set_name_usecase.New(inmemory_player_store.New(), accounts, clock), visits),
	}
}

func TestAKeptNameShowsOnTheRosterAtOnce(t *testing.T) {
	f := setup()
	f.accounts.Link(ada)

	profile, err := f.useCase.Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada_L"})

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada_L"), profile.Name)
	require.Len(t, f.visits.Visits(), 1)
	assert.Equal(t, players.Author{Name: "Ada_L"}, f.visits.Visits()[0].Author)
}

func TestARefusedNameRenamesNothing(t *testing.T) {
	f := setup()

	_, err := f.useCase.Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada_L"})

	require.ErrorIs(t, err, players.ErrNotLinked)
	require.Len(t, f.visits.Visits(), 1)
	assert.Equal(t, guest, f.visits.Visits()[0].Author)
}
