//go:build testing

// Package inmemory_reaction_storage keeps reactions in a slice, for tests that need a reactions.Storage but not
// postgres.
package inmemory_reaction_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func New() *Storage {
	return &Storage{}
}

// Storage holds each reaction put on, oldest first, as postgres holds its rows.
type Storage struct {
	mu    sync.Mutex
	given []reactions.Change
}

var _ reactions.Storage = (*Storage)(nil)

func (s *Storage) Save(_ context.Context, change reactions.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	same := func(each reactions.Change) bool {
		return each.MessageID == change.MessageID && each.Reaction == change.Reaction && each.Reactor == change.Reactor
	}
	if !change.On {
		s.given = slices.DeleteFunc(s.given, same)
		return nil
	}
	if !slices.ContainsFunc(s.given, same) {
		s.given = append(s.given, change)
	}
	return nil
}

func (s *Storage) Reactions(
	_ context.Context,
	ids []messages.MessageID,
) (map[messages.MessageID]reactions.Reactions, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	given := make(map[messages.MessageID]reactions.Reactions)
	for _, change := range s.given {
		if slices.Contains(ids, change.MessageID) {
			given[change.MessageID] = given[change.MessageID].Applied(change)
		}
	}
	return given, nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.given)
	s.given = slices.DeleteFunc(s.given, func(change reactions.Change) bool { return change.At.Before(cutoff) })
	return int64(before - len(s.given)), nil
}
