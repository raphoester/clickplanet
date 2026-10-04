package forget_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase"
)

var (
	ada = standings.AccountID{0: 1, 15: 1}
	bob = standings.AccountID{0: 1, 15: 2}
)

func TestAForgottenAccountLeavesTheStandingsAndTheOthersStay(t *testing.T) {
	store := inmemory_contribution_store.New()
	for _, account := range []standings.AccountID{ada, bob} {
		require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: account, Country: "fr", At: time.Now()}))
	}

	require.NoError(t, forget_account_usecase.New(store).Execute(t.Context(), ada))

	assert.True(t, store.Tally(0, ada).Empty())
	assert.False(t, store.Tally(0, bob).Empty())
}

func TestAFailureToForgetIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, forget_account_usecase.New(store).Execute(t.Context(), ada), refused)
}
