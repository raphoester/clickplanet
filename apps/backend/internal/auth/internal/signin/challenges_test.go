package signin_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func challenges(t *testing.T, offered bool) *signin.Challenges {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	guesses := cpratelimit.New("guesses", cpratelimit.Config{Burst: signin.MaxAttempts, PerSecond: 1 / signin.ChallengeTTL.Seconds()},
		cptime.NewFixedClock(now))
	return signin.NewChallenges(offered, &signin.SequentialSecrets{}, &signin.SequentialCodes{}, sealer, guesses)
}

func TestEveryGuessSpendsOneAndTheRightOneTooOnceTheyAreGone(t *testing.T) {
	c := challenges(t, true)
	challenge, _, err := c.Issued("player@example.com", accounts.IntentSignIn, accounts.AccountID{}, now)
	require.NoError(t, err)

	for range signin.MaxAttempts - 1 {
		require.ErrorIs(t, c.Guess(challenge, "999999", now), signin.ErrWrongCode)
	}
	require.NoError(t, c.Guess(challenge, "000001", now))

	assert.ErrorIs(t, c.Guess(challenge, "000001", now), signin.ErrFlowInvalid)
}
