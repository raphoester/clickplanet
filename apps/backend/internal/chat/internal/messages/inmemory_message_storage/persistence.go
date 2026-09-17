package inmemory_message_storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Persistence interface {
	Insert(ctx context.Context, record messages.Record) error
	Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Load fills the history with the newest messages still within retention.
func (s *Storage) Load(ctx context.Context) error {
	recent, err := s.persistence.Recent(ctx, s.cutoff(), s.config.HistorySize)
	if err != nil {
		return fmt.Errorf("failed to load the chat history: %w", err)
	}

	s.historyMu.Lock()
	s.history = append(s.history[:0], recent...)
	s.historyMu.Unlock()

	s.logger.Info("loaded the chat history", slog.Int("messages", len(recent)))

	return nil
}

func (s *Storage) Name() string { return "chat-storage" }

// Run deletes the messages past retention every PruneInterval.
func (s *Storage) Run(ctx context.Context) {
	s.logger.Info("pruning chat messages", slog.Duration("retention", s.config.Retention))

	prune := time.NewTicker(s.config.PruneInterval)
	defer prune.Stop()

	for {
		select {
		case <-prune.C:
			s.prune(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Storage) prune(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	deleted, err := s.persistence.DeleteBefore(ctx, s.cutoff())
	if err != nil {
		s.logger.Error("failed to prune chat messages, retrying next tick", slog.Any("error", err))
		return
	}

	if deleted > 0 {
		s.logger.Info("pruned chat messages", slog.Int64("deleted", deleted))
	}
}

func (s *Storage) cutoff() time.Time {
	return s.clock.Now().Add(-s.config.Retention)
}
