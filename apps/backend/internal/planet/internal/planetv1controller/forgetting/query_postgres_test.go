package forgetting_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

func TestEachRevertChangesTheForgettingAndNothingElseDoes(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "planet", migrations.FS)
	store, query := postgres_ledger_store.New(db), forgetting.NewPostgresQuery(db)
	forget := func(head ledger.Position, forgotten map[ledger.Caller]ledger.Position) forgetting.Forgetting {
		t.Helper()
		require.NoError(t, store.Save(t.Context(), inmemory_ledger_storage.Changes{
			Entries: slices.Values([]inmemory_ledger_storage.Stored{}),
			Marks:   inmemory_ledger_storage.Marks{Head: head, Forgotten: forgotten},
		}))
		read, err := query.Forgetting(t.Context())
		require.NoError(t, err)
		return read
	}

	none := forget(0, nil)
	scope := forget(0, map[ledger.Caller]ledger.Position{{Scope: "203.0.113.7"}: 10})
	again := forget(0, map[ledger.Caller]ledger.Position{{Scope: "203.0.113.7"}: 10})
	account := forget(0, map[ledger.Caller]ledger.Position{{Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}: 10})
	further := forget(0, map[ledger.Caller]ledger.Position{{Scope: "203.0.113.7"}: 12})

	assert.NotEqual(t, none, scope)
	assert.Equal(t, scope, again, "a revert that forgets nothing more")
	assert.NotEqual(t, scope, account)
	assert.NotEqual(t, account, further)
	assert.NotContains(t, string(scope), "203.0.113.7")
	assert.Equal(t, none, forget(20, nil), "the head past every revert")
}
