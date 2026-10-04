package name_accounts_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

type fixture struct {
	store    *inmemory_player_store.Store
	accounts *titles.FakeAccounts
	useCase  *name_accounts_usecase.UseCase
}

func setup(t *testing.T, names ...players.Name) fixture {
	t.Helper()

	store := inmemory_player_store.New()
	accounts := titles.NewFakeAccounts()
	return fixture{
		store:    store,
		accounts: accounts,
		useCase: name_accounts_usecase.New(store, accounts,
			players.NewGeneratedNames(store, players.NewRepeatedNames(names...)), cptime.NewFixedClock(now)),
	}
}

func (f fixture) player(t *testing.T, account players.AccountID, linked bool) {
	t.Helper()

	require.NoError(t, f.store.RecordTake(t.Context(), account, now))
	f.accounts.Create(account, players.Account{Linked: linked})
}

func TestEveryLinkedAccountWithNoNameIsNamedAndNoOtherOne(t *testing.T) {
	f := setup(t, "BraveFox42", "SlyOtter17")
	f.player(t, players.AccountID{15: 1}, true)
	f.player(t, players.AccountID{15: 2}, false)
	f.player(t, players.AccountID{15: 3}, true)
	require.NoError(t, f.store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 3}, Name: "Ada", UpdatedAt: now}))

	named, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, named)
	profile, err := f.store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.Name)
	_, err = f.store.Profile(t.Context(), players.AccountID{15: 2})
	require.ErrorIs(t, err, players.ErrNoProfile, "a guest is not named")
	profile, err = f.store.Profile(t.Context(), players.AccountID{15: 3})
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.Name)
}

func TestASecondRunNamesNobody(t *testing.T) {
	f := setup(t, "BraveFox42", "SlyOtter17")
	f.player(t, players.AccountID{15: 1}, true)

	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	named, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, named)
}

func TestEveryPageIsRead(t *testing.T) {
	names := make([]players.Name, 0, 501)
	for i := range 501 {
		names = append(names, players.Name("Player"+string(rune('a'+i%26))+string(rune('a'+i/26))))
	}
	f := setup(t, names...)
	for i := range 501 {
		f.player(t, players.AccountID{0: 1, 14: byte(i >> 8), 15: byte(i)}, true)
	}

	named, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 501, named)
}

func TestAFailureToAskAuthNamesNobody(t *testing.T) {
	f := setup(t, "BraveFox42")
	f.player(t, players.AccountID{15: 1}, true)
	refused := errors.New("auth is down")
	f.accounts.FailWith(refused)

	named, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, refused)
	assert.Zero(t, named)
}
