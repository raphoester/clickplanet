//go:build testing

// Package inmemory_message_storage keeps messages in a slice, for tests that need a messages.Storage but not postgres.
package inmemory_message_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu      sync.Mutex
	records []messages.Record
}

var _ messages.Storage = (*Storage)(nil)

func (s *Storage) Append(_ context.Context, record messages.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)
	return nil
}

func (s *Storage) Recent(_ context.Context, since time.Time, limit int) ([]messages.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.recent(since, limit), nil
}

func (s *Storage) Shown(_ context.Context, id messages.MessageID, since time.Time, limit int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.ContainsFunc(s.recent(since, limit), func(message messages.Message) bool {
		return message.ID == id
	}), nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.records)
	s.records = slices.DeleteFunc(s.records, func(record messages.Record) bool {
		return record.Message.SentAt.Before(cutoff)
	})
	return int64(before - len(s.records)), nil
}

func (s *Storage) recent(since time.Time, limit int) []messages.Message {
	var recent []messages.Message
	for _, record := range s.records {
		if !record.Message.SentAt.Before(since) {
			recent = append(recent, record.Message)
		}
	}
	return recent[max(0, len(recent)-limit):]
}
