package resolve_account_usecase_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

type fixture struct {
	useCase  *resolve_account_usecase.UseCase
	sessions *inmemory_account_store.Store
	clock    *cptime.FixedClock
}

func setUp() fixture {
	sessions := inmemory_account_store.New()
	clock := cptime.NewFixedClock(start)
	useCase := resolve_account_usecase.New(
		sessions, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, accounts.Lifetime{}.WithDefaults(), clock)
	return fixture{useCase: useCase, sessions: sessions, clock: clock}
}

func (f fixture) execute(t *testing.T, in resolve_account_usecase.In) *resolve_account_usecase.Out {
	t.Helper()
	out, err := f.useCase.Execute(t.Context(), in)
	require.NoError(t, err)
	return out
}

func cookieOf(t *testing.T, setCookie string) *http.Cookie {
	t.Helper()
	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie
}

func TestNoCookieAndNoAskIsNoAccount(t *testing.T) {
	out, err := setUp().useCase.Execute(t.Context(), resolve_account_usecase.In{})

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Nil(t, out)
}

func TestAskingStartsAGuestAndStoresIt(t *testing.T) {
	f := setUp()

	out := f.execute(t, resolve_account_usecase.In{Create: true})

	assert.Equal(t, uuid.UUID{15: 1}, out.Account)
	assert.Equal(t, "token-1", cookieOf(t, out.SetCookie).Value)
	stored, err := f.sessions.FindSession(t.Context(), accounts.TokenOf("token-1").Hash)
	require.NoError(t, err)
	assert.Equal(t, accounts.StartGuest(uuid.UUID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start), stored)
}

func TestTheCookieBringsBackTheSameAccountWithoutRenewingIt(t *testing.T) {
	f := setUp()
	f.execute(t, resolve_account_usecase.In{Create: true})

	f.clock.Advance(time.Hour)
	out := f.execute(t, resolve_account_usecase.In{CookieHeader: "theme=dark; cp_sid=token-1", Create: true})

	assert.Equal(t, &resolve_account_usecase.Out{Account: uuid.UUID{15: 1}}, out)
}

func TestADayLaterTheExtendedSessionIsSavedAndItsCookieRenewed(t *testing.T) {
	f := setUp()
	f.execute(t, resolve_account_usecase.In{Create: true})

	f.clock.Advance(25 * time.Hour)
	out := f.execute(t, resolve_account_usecase.In{CookieHeader: "cp_sid=token-1"})

	assert.Equal(t, uuid.UUID{15: 1}, out.Account)
	renewed := cookieOf(t, out.SetCookie)
	assert.Equal(t, "token-1", renewed.Value)
	assert.Equal(t, start.Add(25*time.Hour).Add(90*24*time.Hour), renewed.Expires)

	stored, err := f.sessions.FindSession(t.Context(), accounts.TokenOf("token-1").Hash)
	require.NoError(t, err)
	assert.Equal(t, start.Add(25*time.Hour), stored.ExtendedAt)
}

func TestAnExpiredCookieWithoutAskingIsNoAccount(t *testing.T) {
	f := setUp()
	f.execute(t, resolve_account_usecase.In{Create: true})

	f.clock.Advance(91 * 24 * time.Hour)
	out, err := f.useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=token-1"})

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.ErrorIs(t, err, accounts.ErrSessionExpired)
	assert.Nil(t, out)
}

func TestAnExpiredCookieWithAskingIsANewGuest(t *testing.T) {
	f := setUp()
	f.execute(t, resolve_account_usecase.In{Create: true})

	f.clock.Advance(91 * 24 * time.Hour)
	out := f.execute(t, resolve_account_usecase.In{CookieHeader: "cp_sid=token-1", Create: true})

	assert.Equal(t, uuid.UUID{15: 2}, out.Account)
	assert.Equal(t, "token-2", cookieOf(t, out.SetCookie).Value)
}

func TestAnUnknownCookieIsNoAccount(t *testing.T) {
	out, err := setUp().useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=made-up"})

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.ErrorIs(t, err, accounts.ErrSessionNotFound)
	assert.Nil(t, out)
}

func TestAStoreFailureIsAnErrorThatIsNotNoAccount(t *testing.T) {
	f := setUp()
	f.sessions.FailWith(errors.New("postgres is down"))

	out, err := f.useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=abc", Create: true})

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrNoAccount)
	assert.Nil(t, out)
}
