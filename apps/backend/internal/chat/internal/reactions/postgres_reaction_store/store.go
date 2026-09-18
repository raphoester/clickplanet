// Package postgres_reaction_store keeps every chat reaction in chat.reactions: one row per message, reaction and
// reactor. It is the chat's only copy: every read and write goes to postgres.
package postgres_reaction_store

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ reactions.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, change reactions.Change) error {
	if !change.On {
		if _, err := s.db.ExecContext(ctx, `
			DELETE FROM reactions WHERE message_id = $1 AND reaction = $2 AND reactor = $3
		`, string(change.MessageID), int32(change.Reaction), string(change.Reactor)); err != nil {
			return fmt.Errorf("failed to delete a reaction: %w", err)
		}
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO reactions (message_id, reaction, reactor, reacted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, reaction, reactor) DO NOTHING
	`, string(change.MessageID), int32(change.Reaction), string(change.Reactor), change.At.UTC()); err != nil {
		return fmt.Errorf("failed to insert a reaction: %w", err)
	}
	return nil
}

// Reactions replays the rows oldest first, so each reaction keeps the place it first appeared in.
func (s *Store) Reactions(
	ctx context.Context,
	ids []messages.MessageID,
) (map[messages.MessageID]reactions.Reactions, error) {
	// lib/pq cannot take a named array.
	plain := make([]string, 0, len(ids))
	for _, id := range ids {
		plain = append(plain, string(id))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, reaction, reactor
		FROM reactions
		WHERE message_id = ANY($1::text[])
		ORDER BY reacted_at, message_id, reaction, reactor
	`, pq.Array(plain))
	if err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	given := make(map[messages.MessageID]reactions.Reactions)
	for rows.Next() {
		var (
			messageID string
			reaction  int32
			reactor   string
		)
		if err := rows.Scan(&messageID, &reaction, &reactor); err != nil {
			return nil, fmt.Errorf("failed to scan a reaction: %w", err)
		}
		id := messages.MessageID(messageID)
		given[id] = given[id].With(reactions.Reaction(reaction), reactions.Reactor(reactor))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}

	return given, nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM reactions WHERE reacted_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old reactions: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted reactions: %w", err)
	}
	return deleted, nil
}
