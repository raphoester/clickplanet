package start_email_sign_in_usecase_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/usecases/start_email_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

type refusingAttester struct{}

func (refusingAttester) Attest(context.Context, string, string) error {
	return errors.New("siteverify said no")
}

type fixture struct {
	sealer *aes_flow_sealer.Sealer
	store  *inmemory_account_store.Store
	mailer *signin.FakeMailer
	clock  *cptime.FixedClock
}

func setUp(t *testing.T) *fixture {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	return &fixture{sealer: sealer, store: inmemory_account_store.New(), mailer: &signin.FakeMailer{}, clock: cptime.NewFixedClock(start)}
}

func (f *fixture) useCase(offered bool, attester attestation.Attester) *start_email_sign_in_usecase.UseCase {
	sends := cpratelimit.New("sends", cpratelimit.Config{Burst: 3, PerSecond: 1.0 / 1200}, f.clock)
	return start_email_sign_in_usecase.New(offered, attester, signin.BlockedDomains{"mailinator.com"}, f.store, sends,
		&signin.SequentialSecrets{}, &signin.SequentialCodes{}, f.sealer, f.mailer, f.clock)
}

func in(address string) start_email_sign_in_usecase.In {
	return start_email_sign_in_usecase.In{Address: address, AttestationToken: "widget", IP: "192.0.2.1"}
}

func (f *fixture) opened(t *testing.T, setCookie string) *signin.Challenge {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	require.Equal(t, signin.ChallengeCookieName, cookie.Name)
	challenge, err := f.sealer.OpenedChallenge(cookie.Value)
	require.NoError(t, err)
	return challenge
}

func TestTheCodeIsSentToTheAddressAndSealedInTheCookie(t *testing.T) {
	f := setUp(t)

	out, err := f.useCase(true, open_attester.New()).Execute(t.Context(), in(" Player@Example.com "))
	require.NoError(t, err)

	assert.Equal(t, []signin.Sent{{To: "player@example.com", Letter: signin.CodeLetter("000001")}}, f.mailer.Sent())
	assert.Equal(t, &signin.Challenge{
		ID: "secret-1", Address: "player@example.com", Code: "000001", ExpiresAt: start.Add(10 * time.Minute), Intent: accounts.IntentSignIn,
	}, f.opened(t, out.SetCookie))
}

func TestALinkRemembersTheAccountItStartedOn(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.store.CreateGuest(t.Context(), accounts.GuestSession(accounts.AccountID{15: 7}, accounts.TokenOf("guest"), lifetime, start)))
	link := in("player@example.com")
	link.Intent = accounts.IntentLink
	link.CookieHeader = "cp_sid=guest"

	out, err := f.useCase(true, open_attester.New()).Execute(t.Context(), link)
	require.NoError(t, err)

	challenge := f.opened(t, out.SetCookie)
	assert.Equal(t, accounts.IntentLink, challenge.Intent)
	assert.Equal(t, accounts.AccountID{15: 7}, challenge.Account)
}

func TestALinkWithNoAccountSendsNothing(t *testing.T) {
	f := setUp(t)
	link := in("player@example.com")
	link.Intent = accounts.IntentLink

	_, err := f.useCase(true, open_attester.New()).Execute(t.Context(), link)

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Empty(t, f.mailer.Sent())
}

func TestARefusedAddressSendsNothing(t *testing.T) {
	for address, want := range map[string]error{
		"not an address":              signin.ErrAddressInvalid,
		"player@mailinator.com":       signin.ErrAddressDisposable,
		"player@inbox.Mailinator.com": signin.ErrAddressDisposable,
	} {
		t.Run(address, func(t *testing.T) {
			f := setUp(t)
			blocklist := signin.BlockedDomains{"mailinator.com", "inbox.mailinator.com"}
			useCase := start_email_sign_in_usecase.New(true, open_attester.New(), blocklist, f.store,
				cpratelimit.New("sends", cpratelimit.Config{}, f.clock), &signin.SequentialSecrets{}, &signin.SequentialCodes{}, f.sealer, f.mailer, f.clock)

			_, err := useCase.Execute(t.Context(), in(address))

			require.ErrorIs(t, err, want)
			assert.Empty(t, f.mailer.Sent())
		})
	}
}

func TestACallerThatFailsAttestationSendsNothingAndSpendsNoBudget(t *testing.T) {
	f := setUp(t)
	refused := f.useCase(true, refusingAttester{})
	for range 5 {
		_, err := refused.Execute(t.Context(), in("player@example.com"))
		require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	}
	noAddress := in("player@example.com")
	noAddress.IP = ""
	_, err := f.useCase(true, open_attester.New()).Execute(t.Context(), noAddress)
	require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Empty(t, f.mailer.Sent())

	_, err = f.useCase(true, open_attester.New()).Execute(t.Context(), in("player@example.com"))
	assert.NoError(t, err, "the owner of the address still has its whole budget")
}

func TestAnAddressGetsThreeCodesThenOneEveryTwentyMinutes(t *testing.T) {
	f := setUp(t)
	useCase := f.useCase(true, open_attester.New())
	for range 3 {
		_, err := useCase.Execute(t.Context(), in("player@example.com"))
		require.NoError(t, err)
	}

	_, err := useCase.Execute(t.Context(), in("Player@example.com"))
	require.ErrorIs(t, err, signin.ErrTooManyCodes)
	_, err = useCase.Execute(t.Context(), in("other@example.com"))
	require.NoError(t, err, "another address has its own budget")

	f.clock.Advance(20 * time.Minute)
	_, err = useCase.Execute(t.Context(), in("player@example.com"))
	require.NoError(t, err)
	assert.Len(t, f.mailer.Sent(), 5)
}

func TestAMailerThatFailsSetsNoCookie(t *testing.T) {
	f := setUp(t)
	f.mailer.FailWith(errors.New("cloudflare answered 503"))

	out, err := f.useCase(true, open_attester.New()).Execute(t.Context(), in("player@example.com"))

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestEmailSignInOffSendsNothing(t *testing.T) {
	f := setUp(t)

	_, err := f.useCase(false, open_attester.New()).Execute(t.Context(), in("player@example.com"))

	require.ErrorIs(t, err, signin.ErrSignInOff)
	assert.Empty(t, f.mailer.Sent())
}
