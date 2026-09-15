package resolve_account_usecase_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

type fakeSessions struct {
	byHash map[string]accounts.Session
	err    error
	writes int
}

func newSessions() *fakeSessions {
	return &fakeSessions{byHash: map[string]accounts.Session{}}
}

func (f *fakeSessions) FindSession(_ context.Context, tokenHash []byte) (accounts.Session, bool, error) {
	session, found := f.byHash[string(tokenHash)]
	return session, found, f.err
}

func (f *fakeSessions) ExtendSession(_ context.Context, tokenHash []byte, expiresAt, now time.Time) error {
	f.writes++
	session := f.byHash[string(tokenHash)]
	session.ExtendedAt, session.ExpiresAt = now, expiresAt
	f.byHash[string(tokenHash)] = session
	return f.err
}

func (f *fakeSessions) CreateGuest(_ context.Context, account uuid.UUID, tokenHash []byte, expiresAt, now time.Time) error {
	f.writes++
	f.byHash[string(tokenHash)] = accounts.Session{Account: account, ExtendedAt: now, ExpiresAt: expiresAt}
	return f.err
}

func setUp(sessions *fakeSessions) (*resolve_account_usecase.UseCase, *cptime.FixedClock) {
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
	sessions := newSessions()
	useCase, _ := setUp(sessions)

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
	assert.Zero(t, sessions.writes)
}

func TestAskingCreatesAGuestWithACookie(t *testing.T) {
	useCase, _ := setUp(newSessions())

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, out.Account)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, int((90 * 24 * time.Hour).Seconds()), cookie.MaxAge)
}

func TestTheCookieBringsBackTheSameAccountWithoutAWrite(t *testing.T) {
	sessions := newSessions()
	useCase, clock := setUp(sessions)

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{
		CookieHeader: cookieHeaderFrom(t, created.SetCookie), Create: true,
	})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{Account: created.Account}, out)
	assert.Equal(t, 1, sessions.writes, "no second guest, and no extension within the day")
}

func TestADayLaterTheSessionIsExtendedAndTheCookieRenewed(t *testing.T) {
	sessions := newSessions()
	useCase, clock := setUp(sessions)

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(25 * time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: cookieHeaderFrom(t, created.SetCookie)})
	require.NoError(t, err)

	assert.Equal(t, created.Account, out.Account)
	assert.Equal(t, 2, sessions.writes)

	renewed, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	original, err := http.ParseSetCookie(created.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, original.Value, renewed.Value, "the token stays; only its expiry moves")
	assert.Equal(t, start.Add(25*time.Hour).Add(90*24*time.Hour), renewed.Expires)
}

func TestAnExpiredCookieWithoutAskingIsNoAccount(t *testing.T) {
	useCase, clock := setUp(newSessions())

	created, err := useCase.Execute(t.Context(), resolve_account_usecase.In{Create: true})
	require.NoError(t, err)

	clock.Advance(91 * 24 * time.Hour)
	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: cookieHeaderFrom(t, created.SetCookie)})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
}

func TestAnExpiredCookieWithAskingIsANewGuest(t *testing.T) {
	useCase, clock := setUp(newSessions())

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
	useCase, _ := setUp(newSessions())

	out, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=made-up"})
	require.NoError(t, err)

	assert.Equal(t, resolve_account_usecase.Out{}, out)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	sessions := newSessions()
	sessions.err = errors.New("postgres is down")
	useCase, _ := setUp(sessions)

	_, err := useCase.Execute(t.Context(), resolve_account_usecase.In{CookieHeader: "cp_sid=abc", Create: true})

	assert.Error(t, err)
}
