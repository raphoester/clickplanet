//go:build testing

package inmemory_book_store

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Store struct {
	mu    sync.Mutex
	books map[books.BookID]books.Book
}

var _ books.Books = (*Store)(nil)

func New() *Store {
	return &Store{books: map[books.BookID]books.Book{}}
}

func (s *Store) Find(_ context.Context, id books.BookID) (books.Book, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	book, found := s.books[id]
	return book, found, nil
}

func (s *Store) All(context.Context) ([]books.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	all := make([]books.Book, 0, len(s.books))
	for _, book := range s.books {
		all = append(all, book)
	}
	slices.SortFunc(all, func(a, b books.Book) int { return a.StartsAt.Compare(b.StartsAt) })
	return all, nil
}

func (s *Store) Open(_ context.Context, book books.Book) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, kept := s.books[book.ID]; kept {
		return false, nil
	}
	s.books[book.ID] = book
	return true, nil
}

func (s *Store) Close(_ context.Context, book books.Book) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kept, found := s.books[book.ID]
	if !found || kept.Closed {
		return false, nil
	}
	s.books[book.ID] = book
	return true, nil
}
