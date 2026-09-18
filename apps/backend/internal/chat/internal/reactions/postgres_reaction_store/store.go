// Package postgres_reaction_store keeps every chat reaction in chat.reactions: one row per message, reaction and
// reactor. It is the chat's only copy: every read and write goes to postgres.
package postgres_reaction_store

import (
	"context"
	"database/sql"
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

// bumpVersion follows a statement that answers the message_id of each row it changed, as "changed": no row, no bump.
const bumpVersion = `
	INSERT INTO reaction_versions (message_id, version, changed_at)
	SELECT message_id, 1, $4 FROM changed
	ON CONFLICT (message_id) DO UPDATE SET version = reaction_versions.version + 1, changed_at = EXCLUDED.changed_at
`

// Save writes the reaction and bumps the version in one statement, so no reader sees one without the other.
func (s *Store) Save(ctx context.Context, change reactions.Change) error {
	write := `
		WITH changed AS (
			INSERT INTO reactions (message_id, reaction, reactor, reacted_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (message_id, reaction, reactor) DO NOTHING
			RETURNING message_id
		)` + bumpVersion
	if !change.On {
		write = `
		WITH changed AS (
			DELETE FROM reactions WHERE message_id = $1 AND reaction = $2 AND reactor = $3
			RETURNING message_id
		)` + bumpVersion
	}

	if _, err := s.db.ExecContext(ctx, write,
		string(change.MessageID), int32(change.Reaction), string(change.Reactor), change.At.UTC()); err != nil {
		return fmt.Errorf("failed to save a reaction: %w", err)
	}
	return nil
}

// Reactions is one statement, so the reactions and their version are one snapshot. The rows are replayed oldest
// first, so each reaction keeps the place it first appeared in.
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
		SELECT v.message_id, v.version, r.reaction, r.reactor
		FROM reaction_versions v
		LEFT JOIN reactions r ON r.message_id = v.message_id
		WHERE v.message_id = ANY($1::text[])
		ORDER BY r.reacted_at, r.message_id, r.reaction, r.reactor
	`, pq.Array(plain))
	if err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	given := make(map[messages.MessageID]reactions.Reactions)
	for rows.Next() {
		var (
			messageID string
			version   int64
			reaction  sql.NullInt32
			reactor   sql.NullString
		)
		if err := rows.Scan(&messageID, &version, &reaction, &reactor); err != nil {
			return nil, fmt.Errorf("failed to scan a reaction: %w", err)
		}
		id := messages.MessageID(messageID)
		next := given[id].Versioned(uint64(version)) //nolint:gosec // a count of changes, never negative.
		if reaction.Valid {
			next = next.With(reactions.Reaction(reaction.Int32), reactions.Reactor(reactor.String))
		}
		given[id] = next
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read reactions: %w", err)
	}

	return given, nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	// A version outlives its message's reactions only while one of them does: it changed when the last one did.
	result, err := s.db.ExecContext(ctx, `
		WITH versions AS (DELETE FROM reaction_versions WHERE changed_at < $1)
		DELETE FROM reactions WHERE reacted_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old reactions: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted reactions: %w", err)
	}
	return deleted, nil
}
