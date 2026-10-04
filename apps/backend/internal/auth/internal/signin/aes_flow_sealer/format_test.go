package aes_flow_sealer

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

// The JSON the exported Flow and Challenge sealed: cookies in browsers still hold it.
const (
	keptFlow = `{"Provider":"google","State":"state","Verifier":"verifier","Nonce":"nonce",` +
		`"ExpiresAt":"2026-09-16T12:10:00Z","Intent":1,"Account":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,7]}`
	keptChallenge = `{"ID":"challenge-id","Address":"player@example.com","Code":"123456",` +
		`"ExpiresAt":"2026-09-16T12:10:00Z","Intent":1,"Account":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,7]}`
)

var (
	expiry          = time.Date(2026, 9, 16, 12, 10, 0, 0, time.UTC)
	formatFlow      = signin.FlowOf("google", "state", "verifier", "nonce", expiry, accounts.IntentLink, accounts.AccountID{15: 7})
	formatChallenge = signin.ChallengeOf("challenge-id", "player@example.com", "123456", expiry, accounts.IntentLink, accounts.AccountID{15: 7})
)

func formatSealer(t *testing.T) *Sealer {
	t.Helper()

	sealer, err := New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	return sealer
}

func plainOf(t *testing.T, s *Sealer, sealed string, cookie string) string {
	t.Helper()

	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	require.NoError(t, err)
	plain, err := s.aead.Open(nil, nil, raw, []byte(cookie))
	require.NoError(t, err)
	return string(plain)
}

func sealedOf(s *Sealer, plain string, cookie string) string {
	return base64.RawURLEncoding.EncodeToString(s.aead.Seal(nil, nil, []byte(plain), []byte(cookie)))
}

func TestAFlowIsSealedInTheFormatTheBrowsersHold(t *testing.T) {
	s := formatSealer(t)

	sealed, err := s.Sealed(formatFlow)
	require.NoError(t, err)

	assert.JSONEq(t, keptFlow, plainOf(t, s, sealed, signin.FlowCookieName))
}

func TestAFlowABrowserHoldsOpens(t *testing.T) {
	s := formatSealer(t)

	opened, err := s.Opened(sealedOf(s, keptFlow, signin.FlowCookieName))

	require.NoError(t, err)
	assert.Equal(t, formatFlow, opened)
}

func TestAChallengeIsSealedInTheFormatTheBrowsersHold(t *testing.T) {
	s := formatSealer(t)

	sealed, err := s.SealedChallenge(formatChallenge)
	require.NoError(t, err)

	assert.JSONEq(t, keptChallenge, plainOf(t, s, sealed, signin.ChallengeCookieName))
}

func TestAChallengeABrowserHoldsOpens(t *testing.T) {
	s := formatSealer(t)

	opened, err := s.OpenedChallenge(sealedOf(s, keptChallenge, signin.ChallengeCookieName))

	require.NoError(t, err)
	assert.Equal(t, formatChallenge, opened)
}
