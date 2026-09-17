package sign_out_everywhere_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_everywhere_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func TestEverySessionOfTheAccountEndsAndNoOther(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user"}, accounts.AccountID{15: 1}, start)
	laptop := accounts.LinkedSession(identity.Account, accounts.TokenOf("laptop"), lifetime, start)
	phone := accounts.LinkedSession(identity.Account, accounts.TokenOf("phone"), lifetime, start)
	other := accounts.GuestSession(accounts.AccountID{15: 2}, accounts.TokenOf("other"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{NewAccount: true, Identity: identity, Session: laptop}))
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{Session: phone}))
	require.NoError(t, store.CreateGuest(t.Context(), other))

	events := cpbootstrap.NewRecordedEvents()

	setCookie, err := sign_out_everywhere_usecase.New(store, events, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=laptop")

	require.NoError(t, err)
	assert.Equal(t, accounts.ExpiredSessionCookie(), setCookie)
	for _, gone := range []*accounts.Session{laptop, phone} {
		_, err := store.Session(t.Context(), gone.TokenHash)
		require.ErrorIs(t, err, accounts.ErrSessionNotFound)
	}
	_, err = store.Session(t.Context(), other.TokenHash)
	require.NoError(t, err)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&authv1.SignedOut{AccountId: identity.Account.String()}, events.Published()[0]))
}

func TestNoLiveSessionIsNoAccount(t *testing.T) {
	store := inmemory_account_store.New()
	require.NoError(t, store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), lifetime, start)))
	events := cpbootstrap.NewRecordedEvents()
	useCase := sign_out_everywhere_usecase.New(store, events, cptime.NewFixedClock(start.Add(91*24*time.Hour)))

	_, err := useCase.Execute(t.Context(), "cp_sid=token-1")

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Empty(t, events.Published())
}
