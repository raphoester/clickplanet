package aes_flow_sealer_test

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
)

var flow = &signin.Flow{
	Provider: "google", State: "state", Verifier: "verifier", Nonce: "nonce",
	ExpiresAt: time.Date(2026, 9, 16, 12, 10, 0, 0, time.UTC),
	Intent:    accounts.IntentLink, Account: accounts.AccountID{15: 7},
}

func sealer(t *testing.T, seed byte) *aes_flow_sealer.Sealer {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{seed}, 32))
	require.NoError(t, err)
	return sealer
}

func TestASealedFlowOpensAsItWas(t *testing.T) {
	s := sealer(t, 1)

	sealed, err := s.Sealed(flow)
	require.NoError(t, err)
	opened, err := s.Opened(sealed)

	require.NoError(t, err)
	assert.Equal(t, flow, opened)
}

func TestTheBrowserCannotReadTheVerifier(t *testing.T) {
	sealed, err := sealer(t, 1).Sealed(flow)
	require.NoError(t, err)

	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "verifier")
}

func TestAChangedOrForeignCookieDoesNotOpen(t *testing.T) {
	sealed, err := sealer(t, 1).Sealed(flow)
	require.NoError(t, err)
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 1

	for name, cookie := range map[string]string{
		"changed":        base64.RawURLEncoding.EncodeToString(raw),
		"not base64":     "!!!",
		"empty":          "",
		"another seed's": func() string { other, _ := sealer(t, 2).Sealed(flow); return other }(),
	} {
		t.Run(name, func(t *testing.T) {
			opened, err := sealer(t, 1).Opened(cookie)

			require.ErrorIs(t, err, signin.ErrFlowInvalid)
			assert.Nil(t, opened)
		})
	}
}

func TestAnEmptySeedIsRefused(t *testing.T) {
	_, err := aes_flow_sealer.New(nil)

	assert.Error(t, err)
}
