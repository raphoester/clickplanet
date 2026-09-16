package sign_out_everywhere_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_everywhere_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func TestEverySessionOfTheAccountEndsAndNoOther(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user"}, uuid.UUID{15: 1}, start)
	laptop := accounts.StartLinked(identity.Account, accounts.TokenOf("laptop"), lifetime, start)
	phone := accounts.StartLinked(identity.Account, accounts.TokenOf("phone"), lifetime, start)
	other := accounts.StartGuest(uuid.UUID{15: 2}, accounts.TokenOf("other"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{NewAccount: true, Identity: identity, Session: laptop}))
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{Session: phone}))
	require.NoError(t, store.CreateGuest(t.Context(), other))

	setCookie, err := sign_out_everywhere_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=laptop")

	require.NoError(t, err)
	assert.Equal(t, accounts.ClearCookie(), setCookie)
	for _, gone := range []*accounts.Session{laptop, phone} {
		_, err := store.FindSession(t.Context(), gone.TokenHash)
		require.ErrorIs(t, err, accounts.ErrSessionNotFound)
	}
	_, err = store.FindSession(t.Context(), other.TokenHash)
	assert.NoError(t, err)
}

func TestNoLiveSessionIsNoAccount(t *testing.T) {
	store := inmemory_account_store.New()
	require.NoError(t, store.CreateGuest(t.Context(), accounts.StartGuest(uuid.UUID{15: 1}, accounts.TokenOf("token-1"), lifetime, start)))
	useCase := sign_out_everywhere_usecase.New(store, cptime.NewFixedClock(start.Add(91*24*time.Hour)))

	_, err := useCase.Execute(t.Context(), "cp_sid=token-1")

	assert.ErrorIs(t, err, accounts.ErrNoAccount)
}
