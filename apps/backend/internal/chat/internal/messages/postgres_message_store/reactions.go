package postgres_message_store

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func (s *Store) InsertReaction(ctx context.Context, change messages.ReactionChange) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO reactions (message_id, reaction, reactor, reacted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, reaction, reactor) DO NOTHING
	`, string(change.MessageID), int32(change.Reaction), string(change.Reactor), change.At.UTC()); err != nil {
		return fmt.Errorf("failed to insert a reaction: %w", err)
	}
	return nil
}

func (s *Store) DeleteReaction(ctx context.Context, change messages.ReactionChange) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM reactions WHERE message_id = $1 AND reaction = $2 AND reactor = $3
	`, string(change.MessageID), int32(change.Reaction), string(change.Reactor)); err != nil {
		return fmt.Errorf("failed to delete a reaction: %w", err)
	}
	return nil
}

// Reactions is every reaction on the given messages, oldest first, so they can be replayed in the order given.
func (s *Store) Reactions(ctx context.Context, ids []messages.MessageID) ([]messages.ReactionChange, error) {
	// lib/pq cannot take a named array.
	plain := make([]string, 0, len(ids))
	for _, id := range ids {
		plain = append(plain, string(id))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, reaction, reactor, reacted_at
		FROM reactions
		WHERE message_id = ANY($1::text[])
		ORDER BY reacted_at, message_id, reaction, reactor
	`, pq.Array(plain))
	if err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var given []messages.ReactionChange
	for rows.Next() {
		var (
			messageID string
			reaction  int32
			reactor   string
			change    = messages.ReactionChange{On: true}
		)
		if err := rows.Scan(&messageID, &reaction, &reactor, &change.At); err != nil {
			return nil, fmt.Errorf("failed to scan a reaction: %w", err)
		}
		change.MessageID = messages.MessageID(messageID)
		change.Reaction = messages.Reaction(reaction)
		change.Reactor = messages.Reactor(reactor)
		change.At = change.At.UTC()
		given = append(given, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}

	return given, nil
}
