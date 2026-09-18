//go:build testing

// Package inmemory_reaction_storage keeps reactions in a slice, for tests that need a reactions.Storage but not
// postgres.
package inmemory_reaction_storage

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func New() *Storage {
	return &Storage{versions: make(map[messages.MessageID]version)}
}

// Storage holds each reaction put on, oldest first, and each message's version, as postgres holds its rows.
type Storage struct {
	mu       sync.Mutex
	given    []reactions.Change
	versions map[messages.MessageID]version
}

type version struct {
	number    uint64
	changedAt time.Time
}

var _ reactions.Storage = (*Storage)(nil)

func (s *Storage) Save(_ context.Context, change reactions.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	same := func(each reactions.Change) bool {
		return each.MessageID == change.MessageID && each.Reaction == change.Reaction && each.Reactor == change.Reactor
	}
	if slices.ContainsFunc(s.given, same) == change.On {
		return nil
	}
	if change.On {
		s.given = append(s.given, change)
	} else {
		s.given = slices.DeleteFunc(s.given, same)
	}
	s.versions[change.MessageID] = version{number: s.versions[change.MessageID].number + 1, changedAt: change.At}
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
	for id, version := range s.versions {
		if slices.Contains(ids, id) {
			given[id] = given[id].Versioned(version.number)
		}
	}
	return given, nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.given)
	s.given = slices.DeleteFunc(s.given, func(change reactions.Change) bool { return change.At.Before(cutoff) })
	maps.DeleteFunc(s.versions, func(_ messages.MessageID, version version) bool {
		return version.changedAt.Before(cutoff)
	})
	return int64(before - len(s.given)), nil
}
