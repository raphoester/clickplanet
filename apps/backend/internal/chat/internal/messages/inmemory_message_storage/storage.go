//go:build testing

package inmemory_message_storage

import (
	"context"
	"fmt"
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
	records []messages.RecordSnapshot
}

var _ messages.Storage = (*Storage)(nil)

func (s *Storage) Append(_ context.Context, record messages.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record.TakeSnapshot())
	return nil
}

func (s *Storage) Kept() []messages.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.recent(time.Time{}, len(s.records))
}

func (s *Storage) Shown(_ context.Context, id messages.MessageID, since time.Time, limit int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.ContainsFunc(s.recent(since, limit), func(message messages.Snapshot) bool {
		return message.ID == id
	}), nil
}

func (s *Storage) Message(_ context.Context, id messages.MessageID, since time.Time, limit int) (messages.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, kept := range s.recent(since, limit) {
		if kept.ID != id {
			continue
		}
		// As the table: a kept reply holds the id of the message it answers, not what that one said.
		kept.Quote = messages.Quote{MessageID: kept.Quote.MessageID}
		message, err := kept.Restore()
		if err != nil {
			return messages.Message{}, fmt.Errorf("failed to restore message %q: %w", id, err)
		}
		return message, nil
	}
	return messages.Message{}, messages.ErrNotShown
}

func (s *Storage) LatestAddress(_ context.Context, account messages.AccountID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, record := range slices.Backward(s.records) {
		if record.Message.Account == account {
			return record.IP, nil
		}
	}
	return "", messages.ErrNoMessage
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.records)
	s.records = slices.DeleteFunc(s.records, func(record messages.RecordSnapshot) bool {
		return record.Message.SentAt.Before(cutoff)
	})
	return int64(before - len(s.records)), nil
}

func (s *Storage) recent(since time.Time, limit int) []messages.Snapshot {
	var recent []messages.Snapshot
	for _, record := range s.records {
		if !record.Message.SentAt.Before(since) {
			recent = append(recent, record.Message)
		}
	}
	return recent[max(0, len(recent)-limit):]
}
