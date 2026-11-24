package postgres_renewal_store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/postgres_renewal_store"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "shop", migrations.FS)
	suite.Run(t, &renewals.StoreContractSuite{
		NewStore: func() renewals.Store {
			require.NoError(t, db.Purge(t.Context()))
			return inTx{db: db, store: postgres_renewal_store.New(db)}
		},
	})
}

// The lock is a transaction's: each read runs in one, as a use case's does.
type inTx struct {
	db interface {
		InTx(ctx context.Context, fn func(ctx context.Context) error) error
	}
	store *postgres_renewal_store.Store
}

func (s inTx) Renewal(ctx context.Context, id renewals.SubscriptionID) (renewals.Renewal, error) {
	var renewal renewals.Renewal
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		var err error
		renewal, err = s.store.Renewal(ctx, id)
		return err //nolint:wrapcheck // the store named it.
	})
	return renewal, err
}

func (s inTx) Bought(ctx context.Context, buyer orders.AccountID) ([]renewals.Renewal, error) {
	return s.store.Bought(ctx, buyer) //nolint:wrapcheck // the store named it.
}

func (s inTx) Save(ctx context.Context, renewal renewals.Renewal) error {
	return s.store.Save(ctx, renewal) //nolint:wrapcheck // the store named it.
}
