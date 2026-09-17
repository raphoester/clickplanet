package set_name_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func setUp() (*inmemory_player_store.Store, *set_name_usecase.FakeAccounts) {
	accounts := set_name_usecase.NewFakeAccounts()
	accounts.Link(ada)
	accounts.Link(bob)
	return inmemory_player_store.New(), accounts
}

func TestTheNameIsKeptAsTyped(t *testing.T) {
	store, accounts := setUp()

	profile, err := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada_L"})

	require.NoError(t, err)
	want := players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: now}
	assert.Equal(t, want, profile)
	stored, err := store.Profile(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, want, stored)
}

func TestAnInvalidNameChangesNothingAndAsksNobody(t *testing.T) {
	store, accounts := setUp()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada", UpdatedAt: now}))

	_, err := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now.Add(time.Hour))).
		Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "guest_ada"})

	require.ErrorIs(t, err, players.ErrInvalidName)
	assert.Zero(t, accounts.Asked())
	stored, err := store.Profile(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), stored.Name)
}

func TestAGuestIsNotLinkedAndKeepsNoName(t *testing.T) {
	store, accounts := setUp()
	guest := players.AccountID{15: 9}

	_, err := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: guest, Name: "Ada"})

	require.ErrorIs(t, err, players.ErrNotLinked)
	_, err = store.Profile(t.Context(), guest)
	assert.ErrorIs(t, err, players.ErrNoProfile)
}

func TestANameAnotherPlayerHoldsIsTaken(t *testing.T) {
	store, accounts := setUp()
	useCase := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now))
	_, err := useCase.Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada"})
	require.NoError(t, err)

	_, err = useCase.Execute(t.Context(), set_name_usecase.In{Account: bob, Name: "ADA"})

	require.ErrorIs(t, err, players.ErrNameTaken)
}

func TestAPlayerSetsItsOwnNameAgainInAnotherCase(t *testing.T) {
	store, accounts := setUp()
	useCase := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now))
	_, err := useCase.Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "ada"})
	require.NoError(t, err)

	profile, err := useCase.Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "ADA"})

	require.NoError(t, err)
	assert.Equal(t, players.Name("ADA"), profile.Name)
}

func TestAFailureToAskAuthIsNotAGuest(t *testing.T) {
	store, accounts := setUp()
	accounts.FailWith(errors.New("auth is off"))

	_, err := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNotLinked)
}

func TestAStoreFailureIsNotATakenName(t *testing.T) {
	store, accounts := setUp()
	store.FailWith(errors.New("postgres is down"))

	_, err := set_name_usecase.New(store, accounts, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: ada, Name: "Ada"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrInvalidName)
	assert.NotErrorIs(t, err, players.ErrNameTaken)
}
