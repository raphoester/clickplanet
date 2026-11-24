package postgres_grant_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	suite.Run(t, &grants.StoreContractSuite{
		NewStore: func() grants.Store {
			require.NoError(t, db.Purge(t.Context()))
			flag := catalog.OpenFlag("r-0000000001", "Rennes", time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC))
			require.NoError(t, postgres_flag_store.New(db).Save(t.Context(), flag))
			return inTx{db: db, store: postgres_grant_store.New(db)}
		},
		Flag: "r-0000000001",
	})
}

// The lock is a transaction's: each call runs in one, as a use case's does.
type inTx struct {
	db interface {
		InTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	store *postgres_grant_store.Store
}

func (s inTx) Grant(ctx context.Context, id grants.GrantID) (grants.Grant, error) {
	var grant grants.Grant
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		var err error
		grant, err = s.store.Grant(ctx, id)
		return err
	})
	return grant, err
}

func (s inTx) Save(ctx context.Context, grant grants.Grant) error {
	return s.db.InTx(ctx, func(ctx context.Context) error { return s.store.Save(ctx, grant) })
}
