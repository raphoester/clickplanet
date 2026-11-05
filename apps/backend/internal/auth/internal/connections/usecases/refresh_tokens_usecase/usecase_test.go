package refresh_tokens_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/refresh_tokens_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	later  = now.Add(time.Hour)
	claude = admintokens.Holder{Account: accounts.AccountID{15: 1}, ClientID: connections.IDOf("the id"), Client: "Claude"}
)

func setUp(t *testing.T) (*refresh_tokens_usecase.UseCase, *admintokens.MemoryStore, *admintokens.Pair) {
	t.Helper()
	issuer := admintokens.NewIssuer(&admintokens.SequentialSecrets{})
	tokens := &admintokens.MemoryStore{}
	first, err := issuer.Connected(claude, now)
	require.NoError(t, err)
	require.NoError(t, tokens.SaveTokens(t.Context(), first.Tokens()...))

	return refresh_tokens_usecase.New(issuer, tokens, cpatomic.FakeTransactioner{}, cptime.NewFixedClock(later)), tokens, first
}

func trade(t *testing.T, useCase *refresh_tokens_usecase.UseCase, secret admintokens.Secret, client string) (*admintokens.Pair, error) {
	t.Helper()
	return useCase.Execute(t.Context(), refresh_tokens_usecase.In{RefreshToken: secret, ClientID: client}) //nolint:wrapcheck // the tests read the sentinel.
}

func TestARefreshTokenIsTradedOnceForTheNextPair(t *testing.T) {
	useCase, tokens, first := setUp(t)

	next, err := trade(t, useCase, first.TakeSnapshot().Refresh, "the id")
	require.NoError(t, err)

	assert.Equal(t, admintokens.Secret("cprt_secret-5"), next.TakeSnapshot().Refresh, "a new refresh token each time")
	kinds := map[admintokens.Kind]int{}
	for _, kept := range tokens.Kept() {
		kinds[kept.Kind]++
		assert.Equal(t, claude, kept.Holder)
	}
	assert.Equal(t, map[admintokens.Kind]int{admintokens.KindAccess: 2, admintokens.KindRefresh: 1, admintokens.KindSpent: 1}, kinds)
}

func TestARefreshTokenTradedTwiceEndsTheWholeConnection(t *testing.T) {
	useCase, tokens, first := setUp(t)
	_, err := trade(t, useCase, first.TakeSnapshot().Refresh, "the id")
	require.NoError(t, err)

	_, err = trade(t, useCase, first.TakeSnapshot().Refresh, "the id")

	require.ErrorIs(t, err, admintokens.ErrReused)
	require.Len(t, tokens.Kept(), 4)
	for _, kept := range tokens.Kept() {
		assert.False(t, later.Before(kept.ExpiresAt), "the thief's pair and the app's both end")
	}
}

func TestARefreshTokenSentByAnotherClientIsSpentAndBuysNothing(t *testing.T) {
	useCase, tokens, first := setUp(t)

	_, err := trade(t, useCase, first.TakeSnapshot().Refresh, "another id")
	require.ErrorIs(t, err, admintokens.ErrUnknownToken)

	_, err = trade(t, useCase, first.TakeSnapshot().Refresh, "the id")
	require.ErrorIs(t, err, admintokens.ErrReused, "whoever sent it should not hold it: the app connects again")
	assert.Len(t, tokens.Kept(), 2)
}
