package inmemory_book_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
)

func TestTheMemoryStoreKeepsTheBooksAsTheContractSays(t *testing.T) {
	suite.Run(t, &books.BooksContractSuite{NewStore: func() books.Books { return inmemory_book_store.New() }})
}
