package inmemory_contribution_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/inmemory_contribution_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	standings.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() standings.Store { return inmemory_contribution_store.New() }
	s.TallyOf = func(store standings.Store, book books.BookID, account standings.AccountID) standings.Tally {
		return store.(*inmemory_contribution_store.Store).Tally(book, account) //nolint:forcetypeassert // NewStore built it.
	}
}
