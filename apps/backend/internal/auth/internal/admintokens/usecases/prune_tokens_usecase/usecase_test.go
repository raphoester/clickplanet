package prune_tokens_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens/usecases/prune_tokens_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePruneDeletesWhatLapsedAsOfNow(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	tokens := &admintokens.MemoryStore{}
	pair, err := admintokens.NewIssuer(&admintokens.SequentialSecrets{}).Pair(admintokens.Holder{Client: "Claude"}, now)
	require.NoError(t, err)
	require.NoError(t, tokens.SaveTokens(t.Context(), pair.TakeSnapshots()))

	require.NoError(t, prune_tokens_usecase.New(tokens, cptime.NewFixedClock(now.Add(admintokens.AccessLifetime))).Execute(t.Context()))

	require.Len(t, tokens.Kept(), 1)
	assert.Equal(t, admintokens.KindRefresh, tokens.Kept()[0].Kind)
}
