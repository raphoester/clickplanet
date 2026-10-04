package get_caller_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_caller_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func setUp(t *testing.T) (*get_caller_usecase.UseCase, *inmemory_account_store.Store, *cptime.FixedClock) {
	t.Helper()

	sessions := inmemory_account_store.New()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, sessions.CreateGuest(t.Context(), guest))

	clock := cptime.NewFixedClock(start)
	return get_caller_usecase.New(sessions, clock), sessions, clock
}

func TestTheCookieGivesItsAccount(t *testing.T) {
	useCase, _, clock := setUp(t)
	clock.Advance(89 * 24 * time.Hour)

	account, err := useCase.Execute(t.Context(), "theme=dark; cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 1}, account)
}

func TestNoLiveSessionIsNoAccount(t *testing.T) {
	for name, cookie := range map[string]string{
		"no cookie": "theme=dark",
		"unknown":   "cp_sid=made-up",
		"expired":   "cp_sid=token-1",
	} {
		t.Run(name, func(t *testing.T) {
			useCase, _, clock := setUp(t)
			clock.Advance(91 * 24 * time.Hour)

			_, err := useCase.Execute(t.Context(), cookie)

			assert.ErrorIs(t, err, accounts.ErrNoAccount)
		})
	}
}

func TestAStoreFailureIsNotNoAccount(t *testing.T) {
	useCase, sessions, _ := setUp(t)
	sessions.FailWith(errors.New("postgres is down"))

	_, err := useCase.Execute(t.Context(), "cp_sid=token-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrNoAccount)
}
