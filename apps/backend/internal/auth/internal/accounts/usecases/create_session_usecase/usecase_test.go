package create_session_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ip = "203.0.113.7"

var start = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

type refusingAttester struct{}

func (refusingAttester) Attest(context.Context, string, string) error {
	return errors.New("siteverify said no")
}

type fixture struct {
	sessions *inmemory_account_store.Store
	signer   *cpsession.Signer
	verifier *cpsession.Verifier
	clock    *cptime.FixedClock
}

func setUp(t *testing.T) fixture {
	t.Helper()

	secret, public := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Secret: secret, TTL: time.Hour})
	require.NoError(t, err)
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	return fixture{
		sessions: inmemory_account_store.New(),
		signer:   signer,
		verifier: verifier,
		clock:    cptime.NewFixedClock(start),
	}
}

func (f fixture) useCase(attester attestation.Attester) *create_session_usecase.UseCase {
	return create_session_usecase.New(attester, f.sessions, &accounts.SequentialIDs{}, &accounts.SequentialTokens{},
		f.signer, accounts.Lifetime{}.WithDefaults(), f.clock)
}

func (f fixture) create(t *testing.T, useCase *create_session_usecase.UseCase, cookieHeader string) *create_session_usecase.Out {
	t.Helper()

	out, err := useCase.Execute(t.Context(), create_session_usecase.In{AttestationToken: "widget", IP: ip, CookieHeader: cookieHeader})
	require.NoError(t, err)
	return out
}

func (f fixture) accountIn(t *testing.T, out *create_session_usecase.Out) uuid.UUID {
	t.Helper()

	claims, err := f.verifier.Verify(out.Token.Value, ip, f.clock.Now())
	require.NoError(t, err)
	return claims.Account
}

func cookieOf(t *testing.T, setCookie string) *http.Cookie {
	t.Helper()

	cookie, err := http.ParseSetCookie(setCookie)
	require.NoError(t, err)
	return cookie
}

func TestANewCallerIsGivenAGuestSignedIntoItsToken(t *testing.T) {
	f := setUp(t)

	out := f.create(t, f.useCase(open_attester.New()), "")

	assert.Equal(t, uuid.UUID{15: 1}, f.accountIn(t, out))
	assert.Equal(t, "token-1", cookieOf(t, out.SetCookie).Value)
	stored, err := f.sessions.FindSession(t.Context(), accounts.TokenOf("token-1").Hash)
	require.NoError(t, err)
	assert.Equal(t, uuid.UUID{15: 1}, stored.Account)
}

func TestAReturningCallerKeepsItsAccountAndItsCookie(t *testing.T) {
	f := setUp(t)
	useCase := f.useCase(open_attester.New())
	f.create(t, useCase, "")

	f.clock.Advance(time.Hour)
	out := f.create(t, useCase, "theme=dark; cp_sid=token-1")

	assert.Equal(t, uuid.UUID{15: 1}, f.accountIn(t, out))
	assert.Empty(t, out.SetCookie, "within the day the cookie needs no change")
}

func TestADayLaterTheSessionIsExtendedAndTheCookieRenewed(t *testing.T) {
	f := setUp(t)
	useCase := f.useCase(open_attester.New())
	f.create(t, useCase, "")

	f.clock.Advance(25 * time.Hour)
	out := f.create(t, useCase, "cp_sid=token-1")

	renewed := cookieOf(t, out.SetCookie)
	assert.Equal(t, "token-1", renewed.Value)
	assert.Equal(t, start.Add(25*time.Hour).Add(90*24*time.Hour), renewed.Expires)
	stored, err := f.sessions.FindSession(t.Context(), accounts.TokenOf("token-1").Hash)
	require.NoError(t, err)
	assert.Equal(t, start.Add(25*time.Hour), stored.ExtendedAt)
}

func TestAnExpiredOrUnknownCookieStartsANewGuest(t *testing.T) {
	for name, cookie := range map[string]string{
		"expired": "cp_sid=token-1",
		"unknown": "cp_sid=made-up",
	} {
		t.Run(name, func(t *testing.T) {
			f := setUp(t)
			useCase := f.useCase(open_attester.New())
			f.create(t, useCase, "")

			f.clock.Advance(91 * 24 * time.Hour)
			out := f.create(t, useCase, cookie)

			assert.Equal(t, uuid.UUID{15: 2}, f.accountIn(t, out))
			assert.Equal(t, "token-2", cookieOf(t, out.SetCookie).Value)
		})
	}
}

func TestARefusedAttestationCreatesNothing(t *testing.T) {
	f := setUp(t)

	out, err := f.useCase(refusingAttester{}).Execute(t.Context(), create_session_usecase.In{AttestationToken: "widget", IP: ip})

	require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Nil(t, out)
	_, err = f.sessions.FindSession(t.Context(), accounts.TokenOf("token-1").Hash)
	assert.ErrorIs(t, err, accounts.ErrSessionNotFound, "a caller that proved nothing never gets an account")
}

func TestACallerWithNoAddressIsRefusedBeforeAttestation(t *testing.T) {
	f := setUp(t)

	out, err := f.useCase(open_attester.New()).Execute(t.Context(), create_session_usecase.In{AttestationToken: "widget"})

	require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Nil(t, out)
}

func TestAStoreFailureFailsTheMint(t *testing.T) {
	f := setUp(t)
	f.sessions.FailWith(errors.New("postgres is down"))

	out, err := f.useCase(open_attester.New()).Execute(t.Context(), create_session_usecase.In{
		AttestationToken: "widget", IP: ip, CookieHeader: "cp_sid=token-1",
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Nil(t, out)
}

func TestALinkedSessionIsExtendedByTheLinkedLifetime(t *testing.T) {
	f := setUp(t)
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user"}, uuid.UUID{15: 9}, start)
	require.NoError(t, f.sessions.SaveSignIn(t.Context(), accounts.SignIn{
		NewAccount: true, Identity: identity,
		Session: accounts.StartLinked(identity.Account, accounts.TokenOf("linked"), accounts.Lifetime{}.WithDefaults(), start),
	}))

	f.clock.Advance(25 * time.Hour)
	out := f.create(t, f.useCase(open_attester.New()), "cp_sid=linked")

	assert.Equal(t, identity.Account, f.accountIn(t, out))
	assert.Equal(t, start.Add(25*time.Hour).Add(30*24*time.Hour), cookieOf(t, out.SetCookie).Expires)
}
