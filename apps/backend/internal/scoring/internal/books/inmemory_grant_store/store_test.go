package inmemory_grant_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_grant_store"
)

func TestTheMemoryStoreKeepsTheGrantsAsTheContractSays(t *testing.T) {
	suite.Run(t, &books.GrantsContractSuite{NewStore: func() books.Grants { return inmemory_grant_store.New() }})
}
