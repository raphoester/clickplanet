package admintokens_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
)

var (
	now    = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	claude = admintokens.Holder{Account: accounts.AccountID{15: 1}, ClientID: "the client", Client: "Claude"}
)

func TestAPairIsAnAccessTokenForAnHourAndARefreshTokenForTheNext(t *testing.T) {
	pair, err := admintokens.NewIssuer(&admintokens.SequentialSecrets{}).Pair(claude, now)
	require.NoError(t, err)

	assert.Equal(t, admintokens.Secret("cpat_secret-1"), pair.Access())
	assert.Equal(t, admintokens.Secret("cprt_secret-2"), pair.Refresh())
	assert.Equal(t, []admintokens.Snapshot{
		{
			Hash: accounts.TokenOf("cpat_secret-1").Hash(), Kind: admintokens.KindAccess, Holder: claude,
			CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		},
		{
			Hash: accounts.TokenOf("cprt_secret-2").Hash(), Kind: admintokens.KindRefresh, Holder: claude,
			CreatedAt: now, ExpiresAt: now.Add(90 * 24 * time.Hour),
		},
	}, pair.TakeSnapshots(), "only the hashes are kept, by the rule a session's is")
}
