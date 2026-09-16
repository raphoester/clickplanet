package create_anonymous_session_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/create_anonymous_session_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/open_attester"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

type refusingAttester struct{}

func (refusingAttester) Attest(context.Context, string, string) error {
	return errors.New("siteverify said no")
}

func setUp(t *testing.T, attester attestation.Attester) (*create_anonymous_session_usecase.UseCase, *cpsession.Verifier) {
	t.Helper()

	secret, public := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Secret: secret, TTL: time.Hour})
	require.NoError(t, err)
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	return create_anonymous_session_usecase.New(attester, signer, cptime.NewFixedClock(now)), verifier
}

func TestAnAttestedCallerIsMintedATokenWithNoAccountBoundToItsAddress(t *testing.T) {
	useCase, verifier := setUp(t, open_attester.New())

	token, err := useCase.Execute(t.Context(), create_anonymous_session_usecase.In{AttestationToken: "widget", IP: "203.0.113.7"})
	require.NoError(t, err)

	assert.Equal(t, now.Add(time.Hour), token.ExpiresAt)
	claims, err := verifier.Verify(token.Value, "203.0.113.7", now)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, claims.Account)

	_, err = verifier.Verify(token.Value, "203.0.113.8", now)
	assert.Error(t, err, "the token is worth nothing from another address")
}

func TestARefusedAttestationMintsNothing(t *testing.T) {
	useCase, _ := setUp(t, refusingAttester{})

	token, err := useCase.Execute(t.Context(), create_anonymous_session_usecase.In{AttestationToken: "widget", IP: "203.0.113.7"})

	require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Nil(t, token)
}

func TestACallerWithNoAddressIsRefused(t *testing.T) {
	useCase, _ := setUp(t, open_attester.New())

	token, err := useCase.Execute(t.Context(), create_anonymous_session_usecase.In{AttestationToken: "widget"})

	require.ErrorIs(t, err, attestation.ErrAttestationFailed)
	assert.Nil(t, token)
}
