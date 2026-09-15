package session_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain/session_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

type fakeAttester struct {
	err    error
	tokens []string
	ips    []string
}

func (a *fakeAttester) Attest(_ context.Context, token string, ip string) error {
	a.tokens = append(a.tokens, token)
	a.ips = append(a.ips, ip)
	return a.err
}

type countingMinter struct {
	signer *cpsession.Signer
	mints  int
}

func (m *countingMinter) Mint(ip string, account uuid.UUID, at time.Time) (cpsession.Token, error) {
	m.mints++
	return m.signer.Mint(ip, account, at)
}

type askedAccount struct {
	cookieHeader string
	create       bool
}

type fakeAccounts struct {
	resolution domain.Resolution
	err        error
	asked      []askedAccount
}

func (a *fakeAccounts) Resolve(_ context.Context, cookieHeader string, create bool) (domain.Resolution, error) {
	a.asked = append(a.asked, askedAccount{cookieHeader: cookieHeader, create: create})
	return a.resolution, a.err
}

var guest = uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")

func newService(t *testing.T, attester domain.Attester, accounts domain.Accounts) (*session_service.Service, *countingMinter) {
	t.Helper()

	signer, err := cpsession.NewSigner(cpsession.Config{Secret: "a-test-secret", TTL: time.Hour})
	require.NoError(t, err)

	minter := &countingMinter{signer: signer}

	return session_service.New(attester, accounts, minter, cptime.NewFixedClock(now)), minter
}

func request(ip string) session_service.Request {
	return session_service.Request{AttestationToken: "a-widget-token", IP: ip}
}

func TestAnAttestedCallerIsMintedATokenBoundToItsAddress(t *testing.T) {
	attester := &fakeAttester{}
	service, minter := newService(t, attester, &fakeAccounts{})

	minted, err := service.Create(t.Context(), request("203.0.113.7"))
	require.NoError(t, err)
	token := minted.Token

	assert.NotEmpty(t, token.Value)
	assert.NotEmpty(t, token.ID)
	assert.Equal(t, now.Add(time.Hour), token.ExpiresAt)
	assert.Equal(t, 1, minter.mints)

	_, err = minter.signer.Verify(token.Value, "203.0.113.7", now)
	require.NoError(t, err)

	_, err = minter.signer.Verify(token.Value, "203.0.113.8", now)
	assert.Error(t, err, "the token is worth nothing from another address")
}

func TestTheAttestationTokenAndTheAddressReachTheAttester(t *testing.T) {
	attester := &fakeAttester{}
	service, _ := newService(t, attester, &fakeAccounts{})

	_, err := service.Create(t.Context(), request("203.0.113.7"))
	require.NoError(t, err)

	assert.Equal(t, []string{"a-widget-token"}, attester.tokens)
	assert.Equal(t, []string{"203.0.113.7"}, attester.ips)
}

func TestARefusedAttestationMintsNothing(t *testing.T) {
	attester := &fakeAttester{err: errors.New("siteverify said no")}
	accounts := &fakeAccounts{}
	service, minter := newService(t, attester, accounts)

	_, err := service.Create(t.Context(), session_service.Request{
		AttestationToken: "a-widget-token", IP: "203.0.113.7", CookieHeader: "cp_sid=abc", CreateAccount: true,
	})

	assert.ErrorIs(t, err, domain.ErrAttestationFailed)
	assert.Zero(t, minter.mints)
	assert.Empty(t, accounts.asked, "a caller that proved nothing never creates an account")
}

func TestACallerWithNoCookieThatAsksForNoAccountIsMintedNone(t *testing.T) {
	accounts := &fakeAccounts{resolution: domain.Resolution{Account: guest}}
	service, minter := newService(t, &fakeAttester{}, accounts)

	minted, err := service.Create(t.Context(), request("203.0.113.7"))
	require.NoError(t, err)

	assert.Empty(t, accounts.asked)
	assert.Empty(t, minted.SetCookie)
	claims, err := minter.signer.Verify(minted.Token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, claims.Account)
}

func TestTheResolvedAccountIsSignedIntoTheTokenAndItsCookieIsPassedOn(t *testing.T) {
	accounts := &fakeAccounts{resolution: domain.Resolution{Account: guest, SetCookie: "cp_sid=new"}}
	service, minter := newService(t, &fakeAttester{}, accounts)

	minted, err := service.Create(t.Context(), session_service.Request{
		AttestationToken: "a-widget-token", IP: "203.0.113.7", CookieHeader: "other=1", CreateAccount: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []askedAccount{{cookieHeader: "other=1", create: true}}, accounts.asked)
	assert.Equal(t, "cp_sid=new", minted.SetCookie)
	claims, err := minter.signer.Verify(minted.Token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, guest, claims.Account)
}

func TestACookieIsResolvedEvenWithoutAskingForAnAccount(t *testing.T) {
	accounts := &fakeAccounts{resolution: domain.Resolution{Account: guest}}
	service, _ := newService(t, &fakeAttester{}, accounts)

	_, err := service.Create(t.Context(), session_service.Request{
		AttestationToken: "a-widget-token", IP: "203.0.113.7", CookieHeader: "cp_sid=abc",
	})
	require.NoError(t, err)

	assert.Equal(t, []askedAccount{{cookieHeader: "cp_sid=abc", create: false}}, accounts.asked)
}

func TestAFailedResolutionMintsNothing(t *testing.T) {
	service, minter := newService(t, &fakeAttester{}, &fakeAccounts{err: errors.New("auth is down")})

	_, err := service.Create(t.Context(), session_service.Request{
		AttestationToken: "a-widget-token", IP: "203.0.113.7", CreateAccount: true,
	})

	require.Error(t, err)
	assert.Zero(t, minter.mints)
}

// Without an address the binding is to the empty string, which every other
// caller would also verify against — one token that works for everyone.
func TestACallerWithNoAddressIsRefusedBeforeAttestation(t *testing.T) {
	attester := &fakeAttester{}
	service, minter := newService(t, attester, &fakeAccounts{})

	_, err := service.Create(t.Context(), request(""))

	assert.ErrorIs(t, err, domain.ErrAttestationFailed)
	assert.Empty(t, attester.tokens, "attestation is not even attempted")
	assert.Zero(t, minter.mints)
}
