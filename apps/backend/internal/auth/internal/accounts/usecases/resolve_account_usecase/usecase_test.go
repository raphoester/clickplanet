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

func setUp(sessions *inmemory_account_store.Store) (*resolve_account_usecase.UseCase, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(start)
	return resolve_account_usecase.New(sessions, accounts.Lifetime{}.WithDefaults(), clock), clock
}

func cookieHeaderFrom(t *testing.T, setCookie string) string {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return "theme=dark; " + cookie.Name + "=" + cookie.Value
}

func TestNoCookieAndNoAskIsNoAccount(t *testing.T) {
	useCase, _ := setUp(inmemory_account_store.New())

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
}

func TestAskingCreatesAGuestWithACookie(t *testing.T) {
	useCase, _ := setUp(inmemory_account_store.New())

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, out.Account)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, int((90 * 24 * time.Hour).Seconds()), cookie.MaxAge)
}

func TestTheCookieBringsBackTheSameAccountWithoutRenewingIt(t *testing.T) {
	useCase, clock := setUp(inmemory_account_store.New())

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{
		CookieHeader: cookieHeaderFrom(t, created.SetCookie), Create: true,
	})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{Account: created.Account}, out, "the same guest, and no renewed cookie within the day")
}

func TestADayLaterTheSessionIsExtendedAndTheCookieRenewed(t *testing.T) {
	useCase, clock := setUp(inmemory_account_store.New())

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(25 * time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: cookieHeaderFrom(t, created.SetCookie)})
	require.NoError(t, err)

	assert.Equal(t, created.Account, out.Account)

	renewed, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	original, err := http.ParseSetCookie(created.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, original.Value, renewed.Value, "the token stays; only its expiry moves")
	assert.Equal(t, start.Add(25*time.Hour).Add(90*24*time.Hour), renewed.Expires)
}

func TestAnExpiredCookieWithoutAskingIsNoAccount(t *testing.T) {
	useCase, clock := setUp(inmemory_account_store.New())

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(91 * 24 * time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: cookieHeaderFrom(t, created.SetCookie)})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
}

func TestAnExpiredCookieWithAskingIsANewGuest(t *testing.T) {
	useCase, clock := setUp(inmemory_account_store.New())

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(91 * 24 * time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{
		CookieHeader: cookieHeaderFrom(t, created.SetCookie), Create: true,
	})
	require.NoError(t, err)

	assert.NotEqual(t, created.Account, out.Account)
	assert.NotEmpty(t, out.SetCookie)
}

func TestAForgedCookieIsNoAccount(t *testing.T) {
	useCase, _ := setUp(inmemory_account_store.New())

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=made-up"})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	sessions := inmemory_account_store.New()
	sessions.FailWith(errors.New("postgres is down"))
	useCase, _ := setUp(sessions)

	_, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=abc", Create: true})

	assert.Error(t, err)
}
