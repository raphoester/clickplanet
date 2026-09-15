package session_service_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

func (m *countingMinter) Mint(ip string, at time.Time) (cpsession.Token, error) {
	m.mints++
	return m.signer.Mint(ip, at)
}

func newService(t *testing.T, attester domain.Attester) (*session_service.Service, *countingMinter) {
	t.Helper()

	signer, err := cpsession.NewSigner(cpsession.Config{Secret: "a-test-secret", TTL: time.Hour})
	require.NoError(t, err)

	minter := &countingMinter{signer: signer}

	return session_service.New(attester, minter, cptime.NewFixedClock(now)), minter
}

func TestAnAttestedCallerIsMintedATokenBoundToItsAddress(t *testing.T) {
	attester := &fakeAttester{}
	service, minter := newService(t, attester)

	token, err := service.Create(t.Context(), "a-widget-token", "203.0.113.7")
	require.NoError(t, err)

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
	service, _ := newService(t, attester)

	_, err := service.Create(t.Context(), "a-widget-token", "203.0.113.7")
	require.NoError(t, err)

	assert.Equal(t, []string{"a-widget-token"}, attester.tokens)
	assert.Equal(t, []string{"203.0.113.7"}, attester.ips)
}

func TestARefusedAttestationMintsNothing(t *testing.T) {
	attester := &fakeAttester{err: errors.New("siteverify said no")}
	service, minter := newService(t, attester)

	_, err := service.Create(t.Context(), "a-widget-token", "203.0.113.7")

	assert.ErrorIs(t, err, domain.ErrAttestationFailed)
	assert.Zero(t, minter.mints)
}

// Without an address the binding is to the empty string, which every other
// caller would also verify against — one token that works for everyone.
func TestACallerWithNoAddressIsRefusedBeforeAttestation(t *testing.T) {
	attester := &fakeAttester{}
	service, minter := newService(t, attester)

	_, err := service.Create(t.Context(), "a-widget-token", "")

	assert.ErrorIs(t, err, domain.ErrAttestationFailed)
	assert.Empty(t, attester.tokens, "attestation is not even attempted")
	assert.Zero(t, minter.mints)
}
