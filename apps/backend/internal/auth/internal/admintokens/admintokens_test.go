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

func connected(t *testing.T) (*admintokens.Issuer, *admintokens.Pair, []*admintokens.Token) {
	t.Helper()
	issuer := admintokens.NewIssuer(&admintokens.SequentialSecrets{})
	pair, err := issuer.Connected(claude, now)
	require.NoError(t, err)

	connection := make([]*admintokens.Token, 0, len(pair.Tokens()))
	for _, kept := range pair.Tokens() {
		token, err := kept.Restore()
		require.NoError(t, err)
		connection = append(connection, token)
	}
	return issuer, pair, connection
}

func TestAConnectionStartsWithAnAccessTokenForAnHourAndARefreshTokenForTheNext(t *testing.T) {
	_, pair, _ := connected(t)

	assert.Equal(t, admintokens.PairSnapshot{Access: "cpat_secret-2", Refresh: "cprt_secret-3"}, pair.TakeSnapshot())
	assert.Equal(t, []admintokens.Snapshot{
		{
			Hash: accounts.TokenOf("cpat_secret-2").Hash(), Kind: admintokens.KindAccess, Holder: claude,
			Connection: "secret-1", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		},
		{
			Hash: accounts.TokenOf("cprt_secret-3").Hash(), Kind: admintokens.KindRefresh, Holder: claude,
			Connection: "secret-1", CreatedAt: now, ExpiresAt: now.Add(90 * 24 * time.Hour),
		},
	}, pair.Tokens(), "only the hashes are kept, by the rule a session's is")
}

func TestATradeSpendsTheRefreshTokenForAPairOfTheSameConnection(t *testing.T) {
	issuer, first, connection := connected(t)
	later := now.Add(time.Hour)

	trade, err := issuer.Trade(connection, first.TakeSnapshot().Refresh, "the client", later)
	require.NoError(t, err)

	require.NoError(t, trade.Refusal)
	spent := first.Tokens()[1]
	spent.Kind = admintokens.KindSpent
	assert.Equal(t, append([]admintokens.Snapshot{spent}, trade.Pair.Tokens()...), trade.Kept)
	assert.Equal(t, admintokens.ConnectionID("secret-1"), trade.Pair.Tokens()[0].Connection)
}

func TestARefreshTokenTradedTwiceEndsItsConnection(t *testing.T) {
	issuer, first, connection := connected(t)
	spent := first.Tokens()[1]
	spent.Kind = admintokens.KindSpent
	token, err := spent.Restore()
	require.NoError(t, err)
	later := now.Add(time.Hour)

	trade, err := issuer.Trade([]*admintokens.Token{connection[0], token}, first.TakeSnapshot().Refresh, "the client", later)
	require.NoError(t, err)

	require.ErrorIs(t, trade.Refusal, admintokens.ErrReused)
	require.ErrorIs(t, trade.Refusal, admintokens.ErrUnknownToken, "the app is told what a lapsed token tells it")
	assert.Nil(t, trade.Pair)
	require.Len(t, trade.Kept, 2)
	for _, ended := range trade.Kept {
		assert.Equal(t, later, ended.ExpiresAt, "a thief holds a copy: every token of the connection ends")
	}
}

func TestARefreshTokenSentByAnotherClientIsSpentAndBuysNothing(t *testing.T) {
	issuer, first, connection := connected(t)

	trade, err := issuer.Trade(connection, first.TakeSnapshot().Refresh, "another client", now)
	require.NoError(t, err)

	require.ErrorIs(t, trade.Refusal, admintokens.ErrUnknownToken)
	assert.Nil(t, trade.Pair)
	require.Len(t, trade.Kept, 1)
	assert.Equal(t, admintokens.KindSpent, trade.Kept[0].Kind)
}

func TestAnAccessTokenALapsedOneOrAStrangerBuysNothingAndChangesNothing(t *testing.T) {
	issuer, first, connection := connected(t)

	for name, asked := range map[string]struct {
		secret admintokens.Secret
		at     time.Time
	}{
		"an access token": {first.TakeSnapshot().Access, now},
		"lapsed":          {first.TakeSnapshot().Refresh, now.Add(admintokens.RefreshLifetime)},
		"a stranger":      {"cprt_made-up", now},
	} {
		trade, err := issuer.Trade(connection, asked.secret, "the client", asked.at)
		require.NoError(t, err, name)
		require.ErrorIs(t, trade.Refusal, admintokens.ErrUnknownToken, name)
		assert.Empty(t, trade.Kept, name)
	}
}

func TestATokenWithNoConnectionOrAnUnknownKindIsNotRestored(t *testing.T) {
	_, first, _ := connected(t)
	for _, broken := range []func(*admintokens.Snapshot){
		func(s *admintokens.Snapshot) { s.Connection = "" },
		func(s *admintokens.Snapshot) { s.Kind = "lost" },
		func(s *admintokens.Snapshot) { s.Hash = nil },
	} {
		kept := first.Tokens()[0]
		broken(&kept)
		_, err := kept.Restore()
		require.ErrorIs(t, err, admintokens.ErrInvalidToken)
	}
}
