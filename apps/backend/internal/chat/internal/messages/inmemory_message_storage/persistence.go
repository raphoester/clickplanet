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

	// InsertReaction records a reaction put on; one already there is not an error.
	InsertReaction(ctx context.Context, change messages.ReactionChange) error
	// DeleteReaction takes a reaction off; one not there is not an error.
	DeleteReaction(ctx context.Context, change messages.ReactionChange) error
	// Reactions is every reaction on the given messages, oldest first, as the changes that put them on.
	Reactions(ctx context.Context, ids []messages.MessageID) ([]messages.ReactionChange, error)
	DeleteReactionsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Load fills the history with the newest messages still within retention, and each with its reactions.
func (s *Storage) Load(ctx context.Context) error {
	recent, err := s.persistence.Recent(ctx, s.cutoff(), s.config.HistorySize)
	if err != nil {
		return fmt.Errorf("failed to load the chat history: %w", err)
	}

	ids := make([]messages.MessageID, 0, len(recent))
	for _, message := range recent {
		ids = append(ids, message.ID)
	}

	given, err := s.persistence.Reactions(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to load the chat reactions: %w", err)
	}

	reactions := make(map[messages.MessageID]messages.Reactions, len(recent))
	for _, change := range given {
		reactions[change.MessageID] = reactions[change.MessageID].Applied(change)
	}

	s.historyMu.Lock()
	s.history = append(s.history[:0], recent...)
	s.reactions = reactions
	s.historyMu.Unlock()

	s.logger.Info("loaded the chat history", slog.Int("messages", len(recent)), slog.Int("reactions", len(given)))

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

	// A reaction is never older than its message, so what outlives a pruned message is gone a little later.
	deleted, err = s.persistence.DeleteReactionsBefore(ctx, s.cutoff())
	if err != nil {
		s.logger.Error("failed to prune chat reactions, retrying next tick", slog.Any("error", err))
		return
	}

	if deleted > 0 {
		s.logger.Info("pruned chat reactions", slog.Int64("deleted", deleted))
	}
}

func (s *Storage) cutoff() time.Time {
	return s.clock.Now().Add(-s.config.Retention)
}
