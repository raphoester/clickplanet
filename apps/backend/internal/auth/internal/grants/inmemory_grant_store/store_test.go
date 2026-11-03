package inmemory_grant_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/inmemory_grant_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &grants.StoreContractSuite{
		NewStore:   func() grants.Store { return inmemory_grant_store.New() },
		Kept:       func(store grants.Store) []grants.Snapshot { return store.(*inmemory_grant_store.Store).Kept() },
		NewAccount: func(grants.AccountID) {},
	})
}
