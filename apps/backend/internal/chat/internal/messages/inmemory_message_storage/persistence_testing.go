//go:build testing

package inmemory_message_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// MemoryPersistence is a Persistence held in a slice, for tests that need a storage but not postgres.
type MemoryPersistence struct {
	mu        sync.Mutex
	rows      []messages.Record
	reactions []messages.ReactionChange
	failing   error
}

func NewMemoryPersistence(rows ...messages.Record) *MemoryPersistence {
	return &MemoryPersistence{rows: slices.Clone(rows)}
}

// WithReactions is m holding these reactions too, as if recorded before a boot.
func (m *MemoryPersistence) WithReactions(given ...messages.ReactionChange) *MemoryPersistence {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reactions = append(m.reactions, given...)
	return m
}

func (m *MemoryPersistence) Insert(_ context.Context, record messages.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.rows = append(m.rows, record)
	return nil
}

func (m *MemoryPersistence) Recent(_ context.Context, since time.Time, limit int) ([]messages.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var recent []messages.Message
	for _, row := range m.rows {
		if !row.Message.SentAt.Before(since) {
			recent = append(recent, row.Message)
		}
	}
	return recent[max(0, len(recent)-limit):], nil
}

func (m *MemoryPersistence) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return 0, m.failing
	}
	before := len(m.rows)
	m.rows = slices.DeleteFunc(m.rows, func(row messages.Record) bool { return row.Message.SentAt.Before(cutoff) })
	return int64(before - len(m.rows)), nil
}

func (m *MemoryPersistence) InsertReaction(_ context.Context, change messages.ReactionChange) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	if slices.ContainsFunc(m.reactions, sameReaction(change)) {
		return nil
	}
	m.reactions = append(m.reactions, change)
	return nil
}

func (m *MemoryPersistence) DeleteReaction(_ context.Context, change messages.ReactionChange) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.reactions = slices.DeleteFunc(m.reactions, sameReaction(change))
	return nil
}

func (m *MemoryPersistence) Reactions(
	_ context.Context,
	ids []messages.MessageID,
) ([]messages.ReactionChange, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var given []messages.ReactionChange
	for _, change := range m.reactions {
		if slices.Contains(ids, change.MessageID) {
			given = append(given, change)
		}
	}
	return given, nil
}

func (m *MemoryPersistence) DeleteReactionsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return 0, m.failing
	}
	before := len(m.reactions)
	m.reactions = slices.DeleteFunc(m.reactions, func(change messages.ReactionChange) bool {
		return change.At.Before(cutoff)
	})
	return int64(before - len(m.reactions)), nil
}

// StoredReactions is every reaction recorded, oldest first.
func (m *MemoryPersistence) StoredReactions() []messages.ReactionChange {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.reactions)
}

func sameReaction(change messages.ReactionChange) func(messages.ReactionChange) bool {
	return func(each messages.ReactionChange) bool {
		return each.MessageID == change.MessageID && each.Reaction == change.Reaction && each.Reactor == change.Reactor
	}
}

func (m *MemoryPersistence) Stored() []messages.Record {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.rows)
}

// FailWith makes every call return err until Heal.
func (m *MemoryPersistence) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

func (m *MemoryPersistence) Heal() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = nil
}
