//go:build testing

package inmemory_contribution_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type key struct {
	book    books.BookID
	account standings.AccountID
}

type Store struct {
	mu       sync.Mutex
	tallies  map[key]standings.Tally
	failWith error
}

var _ standings.Store = (*Store)(nil)

func New() *Store {
	return &Store{tallies: map[key]standings.Tally{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) RecordTake(_ context.Context, book books.BookID, take standings.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	at := key{book: book, account: take.Account}
	s.tallies[at] = s.tallies[at].WithTake(take)
	return nil
}

func (s *Store) Tally(book books.BookID, account standings.AccountID) standings.Tally {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tallies[key{book: book, account: account}]
}

func (s *Store) DeleteAccount(_ context.Context, account standings.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for at := range s.tallies {
		if at.account == account {
			delete(s.tallies, at)
		}
	}
	return nil
}
