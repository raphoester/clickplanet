package anonymize_takes_usecase_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/anonymize_takes_usecase"
)

var at = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

const kept = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"

func loaded(t *testing.T, persistence *inmemory_ledger_storage.MemoryPersistence) *inmemory_ledger_storage.Storage {
	t.Helper()
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, persistence, slog.New(slog.DiscardHandler))
	require.NoError(t, takings.Load(t.Context()))
	return takings
}

func gone(t *testing.T) ledger.AccountID {
	t.Helper()
	account, err := ledger.AccountIDOf("0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
	require.NoError(t, err)
	return account
}

func TestATakeNotYetFlushedIsKeptWithNoAccount(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	takings := loaded(t, persistence)
	takings.Append(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: gone(t).String(), Country: "fr", At: at})
	takings.Append(ledger.Taking{Tile: 2, Scope: "1.2.3.4", Account: kept, Country: "de", Previous: "fr", At: at})

	require.NoError(t, anonymize_takes_usecase.New(takings, persistence).Execute(t.Context(), gone(t)))

	assert.Equal(t, []inmemory_ledger_storage.Stored{
		{Position: 0, Taking: ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", At: at}},
		{Position: 1, Taking: ledger.Taking{Tile: 2, Scope: "1.2.3.4", Account: kept, Country: "de", Previous: "fr", At: at}},
	}, persistence.Stored())
}

func TestAFailedFlushIsAnError(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	takings := loaded(t, persistence)
	takings.Append(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: gone(t).String(), Country: "fr", At: at})
	persistence.FailWith(errors.New("postgres is down"))

	assert.ErrorContains(t, anonymize_takes_usecase.New(takings, persistence).Execute(t.Context(), gone(t)), "postgres is down")
}
