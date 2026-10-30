package complete_crazygames_sign_in_usecase_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/complete_crazygames_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

type fixture struct {
	crazyGames *signin.FakeTokenProvider
	store      *inmemory_account_store.Store
	arrivals   *inmemory_arrival_store.Store
	outbox     *cpatomic.FakeOutbox
	useCase    *complete_crazygames_sign_in_usecase.UseCase
}

func setUp(t *testing.T, providers func(*signin.FakeTokenProvider) signin.TokenProviders) *fixture {
	t.Helper()

	f := &fixture{
		crazyGames: signin.NewFakeTokenProvider(signin.CrazyGames),
		store:      inmemory_account_store.New(),
		arrivals:   inmemory_arrival_store.New(),
		outbox:     &cpatomic.FakeOutbox{},
	}
	f.crazyGames.Sign("nova's token", accounts.ClaimOf("cg-nova", "", false))
	f.crazyGames.Sign("vega's token", accounts.ClaimOf("cg-vega", "", false))
	register := arrivals.NewRegister(f.arrivals, &arrivals.SequentialIDs{})
	admitter := signin.NewAdmitter(cpatomic.FakeTransactioner{}, f.store, f.outbox, register, &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime)
	f.useCase = complete_crazygames_sign_in_usecase.New(providers(f.crazyGames), admitter, cptime.NewFixedClock(start))
	return f
}

func offered(provider *signin.FakeTokenProvider) signin.TokenProviders {
	return signin.TokenProviders{signin.CrazyGames: provider}
}

func (f *fixture) guest(t *testing.T, account byte, token string) {
	t.Helper()

	require.NoError(t, f.store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: account}, accounts.TokenOf(token), lifetime, start)))
}

func (f *fixture) signIn(t *testing.T, token string, cookies string) (*complete_crazygames_sign_in_usecase.Out, error) {
	t.Helper()

	return f.useCase.Execute(t.Context(), complete_crazygames_sign_in_usecase.In{
		Token: token, CookieHeader: cookies, Landing: arrivals.Landing{Tag: "cg", Referrer: "www.crazygames.com"},
	})
}

func (f *fixture) sessionAccount(t *testing.T, setCookie string) accounts.AccountID {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	session, err := f.store.Session(t.Context(), accounts.TokenOf(cookie.Value).Hash())
	require.NoError(t, err)
	return session.Account()
}

func TestAGuestKeepsItsAccountAndItsProgress(t *testing.T) {
	f := setUp(t, offered)
	f.guest(t, 7, "guest")

	out, err := f.signIn(t, "nova's token", "cp_sid=guest")
	require.NoError(t, err)

	assert.Equal(t, accounts.AccountID{15: 7}, out.Account())
	assert.Equal(t, accounts.Linked, out.Outcome(), "the client spends the first sign-in's refill on this outcome")
	assert.Equal(t, out.Account(), f.sessionAccount(t, out.SetCookie()), "the next mint carries the account linked")
	identity, err := f.store.Identity(t.Context(), signin.CrazyGames, "cg-nova")
	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 7}, identity.Account())
	assert.Empty(t, identity.Email())
	require.Len(t, f.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&authv1.SignedIn{AccountId: out.Account().String(), PreviousAccountId: out.Account().String()}, f.outbox.Recorded()[0]))
	kept, err := f.arrivals.Of(t.Context(), out.Account())
	require.NoError(t, err)
	assert.Empty(t, kept, "the guest's arrival was kept when the guest was made")
}

func TestAPlayerBackOnAnotherDeviceSignsInToItsAccount(t *testing.T) {
	f := setUp(t, offered)
	f.guest(t, 7, "first device")
	first, err := f.signIn(t, "nova's token", "cp_sid=first device")
	require.NoError(t, err)
	f.guest(t, 9, "second device")

	out, err := f.signIn(t, "nova's token", "cp_sid=second device")
	require.NoError(t, err)

	assert.Equal(t, first.Account(), out.Account())
	assert.Equal(t, accounts.SignedIn, out.Outcome())
	assert.Equal(t, first.Account(), f.sessionAccount(t, out.SetCookie()))
}

func TestEveryPageLoadSignsInAgainOnTheSameAccount(t *testing.T) {
	f := setUp(t, offered)
	first, err := f.signIn(t, "nova's token", "")
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(first.SetCookie())
	require.NoError(t, err)

	out, err := f.signIn(t, "nova's token", "cp_sid="+cookie.Value)
	require.NoError(t, err)

	assert.Equal(t, first.Account(), out.Account())
	assert.Equal(t, accounts.SignedIn, out.Outcome())
}

func TestANewPlayerWithNoGuestGetsAnAccountAndItsArrival(t *testing.T) {
	f := setUp(t, offered)

	out, err := f.signIn(t, "nova's token", "")
	require.NoError(t, err)

	assert.Equal(t, accounts.Created, out.Outcome())
	kept, err := f.arrivals.Of(t.Context(), out.Account())
	require.NoError(t, err)
	require.Len(t, kept, 1)
	assert.Equal(t, arrivals.SourceCrazyGames, kept[0].TakeSnapshot().Source)
	assert.Equal(t, "www.crazygames.com", kept[0].TakeSnapshot().Referrer)
}

func TestAnotherPlayerOnASharedDeviceGetsAnAccountOfItsOwn(t *testing.T) {
	f := setUp(t, offered)
	nova, err := f.signIn(t, "nova's token", "")
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(nova.SetCookie())
	require.NoError(t, err)

	vega, err := f.signIn(t, "vega's token", "cp_sid="+cookie.Value)
	require.NoError(t, err)

	assert.NotEqual(t, nova.Account(), vega.Account())
	assert.Equal(t, accounts.Created, vega.Outcome())
}

func TestATokenCrazyGamesDidNotSignIsRefusedAndWritesNothing(t *testing.T) {
	f := setUp(t, offered)
	f.guest(t, 7, "guest")

	_, err := f.signIn(t, "forged", "cp_sid=guest")

	require.ErrorIs(t, err, signin.ErrProviderRefused)
	_, err = f.store.Identity(t.Context(), signin.CrazyGames, "forged")
	require.ErrorIs(t, err, accounts.ErrIdentityNotFound)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAPortalThatCannotBeReachedSignsNobodyIn(t *testing.T) {
	f := setUp(t, offered)
	unreachable := errors.New("the portal is down")
	f.crazyGames.FailWith(unreachable)

	_, err := f.signIn(t, "nova's token", "")

	require.ErrorIs(t, err, unreachable)
	assert.Empty(t, f.outbox.Recorded())
}

func TestCrazyGamesNotOfferedIsSignInOff(t *testing.T) {
	f := setUp(t, func(*signin.FakeTokenProvider) signin.TokenProviders { return signin.TokenProviders{} })

	_, err := f.signIn(t, "nova's token", "")

	require.ErrorIs(t, err, signin.ErrSignInOff)
}
