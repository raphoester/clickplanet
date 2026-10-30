package anonymize_arrivals_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/usecases/anonymize_arrivals_usecase"
)

var (
	at  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	ada = arrivals.AccountID{15: 1}
	bob = arrivals.AccountID{15: 2}
)

func TestTheAccountsArrivalKeepsItsSourceAndLosesItsAccount(t *testing.T) {
	store := inmemory_arrival_store.New()
	kept := arrivals.Snapshot{ID: arrivals.ArrivalID{15: 2}, Account: bob, Source: arrivals.SourceYouTube, At: at}
	store.Seed(arrivals.Snapshot{ID: arrivals.ArrivalID{15: 1}, Account: ada, Source: arrivals.SourceRecruit, Flag: "fr", At: at}, kept)

	require.NoError(t, anonymize_arrivals_usecase.New(store).Execute(t.Context(), ada))

	assert.ElementsMatch(t, []arrivals.Snapshot{
		{ID: arrivals.ArrivalID{15: 1}, Source: arrivals.SourceRecruit, Flag: "fr", At: at},
		kept,
	}, store.Kept())
}

func TestAnAccountWithNoArrivalChangesNothing(t *testing.T) {
	store := inmemory_arrival_store.New()
	kept := arrivals.Snapshot{ID: arrivals.ArrivalID{15: 1}, Account: bob, Source: arrivals.SourceUnknown, At: at}
	store.Seed(kept)

	require.NoError(t, anonymize_arrivals_usecase.New(store).Execute(t.Context(), ada))

	assert.Equal(t, []arrivals.Snapshot{kept}, store.Kept())
}

func TestAFailureToAnonymizeIsAnError(t *testing.T) {
	store := inmemory_arrival_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	assert.ErrorIs(t, anonymize_arrivals_usecase.New(store).Execute(t.Context(), ada), refused)
}
