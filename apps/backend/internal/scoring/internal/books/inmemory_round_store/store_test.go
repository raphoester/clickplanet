package inmemory_round_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
)

func TestTheMemoryStoreKeepsTheRoundsAsTheContractSays(t *testing.T) {
	suite.Run(t, &books.RoundsContractSuite{NewStores: func() (books.Books, books.Rounds) {
		return inmemory_book_store.New(), inmemory_round_store.New()
	}})
}

func TestTheMemoryStoreCountsTheTakesAsTheContractSays(t *testing.T) {
	var store *inmemory_round_store.Store
	suite.Run(t, &books.TakesContractSuite{
		NewStores: func() (books.Books, books.Rounds, books.Takes) {
			store = inmemory_round_store.New()
			return inmemory_book_store.New(), store, store
		},
		CountedIn: func(round books.RoundID) books.Counted { return store.CountedIn(round) },
	})
}
