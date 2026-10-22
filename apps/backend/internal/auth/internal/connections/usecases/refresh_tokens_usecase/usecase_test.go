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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	claude = admintokens.Holder{Account: accounts.AccountID{15: 1}, ClientID: connections.IDOf("the id"), Client: "Claude"}
)

func setUp(t *testing.T) (*refresh_tokens_usecase.UseCase, *admintokens.MemoryStore, *admintokens.Pair) {
	t.Helper()
	issuer := admintokens.NewIssuer(&admintokens.SequentialSecrets{})
	tokens := &admintokens.MemoryStore{}
	first, err := issuer.Pair(claude, now)
	require.NoError(t, err)
	require.NoError(t, tokens.SaveTokens(t.Context(), first.TakeSnapshots()))

	return refresh_tokens_usecase.New(issuer, tokens, cptime.NewFixedClock(now.Add(time.Hour))), tokens, first
}

func TestARefreshTokenIsTradedOnceForTheNextPair(t *testing.T) {
	useCase, tokens, first := setUp(t)

	next, err := useCase.Execute(t.Context(), refresh_tokens_usecase.In{RefreshToken: first.Refresh(), ClientID: "the id"})
	require.NoError(t, err)
	assert.Equal(t, admintokens.Secret("cprt_secret-4"), next.Refresh(), "a new refresh token each time")
	assert.Len(t, tokens.Kept(), 3, "the first access token, and the new pair")
	for _, kept := range tokens.Kept() {
		assert.Equal(t, claude, kept.Holder)
	}

	_, err = useCase.Execute(t.Context(), refresh_tokens_usecase.In{RefreshToken: first.Refresh(), ClientID: "the id"})
	require.ErrorIs(t, err, admintokens.ErrUnknownToken, "spent")
}

func TestARefreshTokenSentByAnotherClientIsSpentAndBuysNothing(t *testing.T) {
	useCase, tokens, first := setUp(t)

	_, err := useCase.Execute(t.Context(), refresh_tokens_usecase.In{RefreshToken: first.Refresh(), ClientID: "another id"})
	require.ErrorIs(t, err, admintokens.ErrUnknownToken)

	_, err = useCase.Execute(t.Context(), refresh_tokens_usecase.In{RefreshToken: first.Refresh(), ClientID: "the id"})
	require.ErrorIs(t, err, admintokens.ErrUnknownToken, "whoever sent it should not hold it: the app connects again")
	assert.Len(t, tokens.Kept(), 1)
}
