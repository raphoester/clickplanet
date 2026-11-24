package postgres_grant_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// "grnt": the flags module's advisory locks are its own, by schema; this one names a payment.
const grantLock = 0x67726e74

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ grants.Store = (*Store)(nil)

func (s *Store) Grant(ctx context.Context, id grants.GrantID) (grants.Grant, error) {
	tx := s.db.On(ctx)
	// A grant and its taking back come on two streams: the second waits for the first's transaction.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, grantLock, id.String()); err != nil {
		return grants.Grant{}, fmt.Errorf("failed to lock the grant: %w", err)
	}
	kept := grants.Snapshot{ID: id}
	var submission uuid.UUID
	err := tx.QueryRowContext(ctx, `SELECT submission_id, period, held_at FROM time_held WHERE grant_id = $1`,
		id.String()).Scan(&submission, &kept.Period, &kept.HeldAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return grants.Grant{}, fmt.Errorf("failed to read whether %s is held: %w", id, err)
	default:
		kept.Held, kept.Submission, kept.HeldAt = true, submissions.SubmissionID(submission), kept.HeldAt.UTC()
	}
	var flag string
	var addedMicros int64
	err = tx.QueryRowContext(ctx, `
		SELECT flag, (extract(epoch FROM added) * 1000000)::bigint, granted_at FROM time_grants WHERE id = $1
	`, id.String()).Scan(&flag, &addedMicros, &kept.GrantedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return grants.Grant{}, fmt.Errorf("failed to read the grant %s: %w", id, err)
	default:
		kept.Granted, kept.Flag = true, catalog.FlagID(flag)
		kept.Added, kept.GrantedAt = time.Duration(addedMicros)*time.Microsecond, kept.GrantedAt.UTC()
	}
	err = tx.QueryRowContext(ctx, `SELECT decided_at FROM time_takebacks WHERE grant_id = $1`, id.String()).
		Scan(&kept.TakenBackAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return grants.Grant{}, fmt.Errorf("failed to read whether %s was taken back: %w", id, err)
	default:
		kept.TakenBack, kept.TakenBackAt = true, kept.TakenBackAt.UTC()
	}
	if !kept.Held && !kept.Granted && !kept.TakenBack {
		return grants.Unheard(id), nil
	}
	grant, err := kept.Restore()
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to restore a grant kept: %w", err)
	}
	return grant, nil
}

func (s *Store) Held(ctx context.Context, submission submissions.SubmissionID) ([]grants.Grant, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT h.grant_id FROM time_held h
		WHERE h.submission_id = $1
		  AND NOT EXISTS (SELECT 1 FROM time_grants g WHERE g.id = h.grant_id)
		  AND NOT EXISTS (SELECT 1 FROM time_takebacks t WHERE t.grant_id = h.grant_id)
		ORDER BY h.held_at, h.grant_id
	`, uuid.UUID(submission))
	if err != nil {
		return nil, fmt.Errorf("failed to list the grants held for %s: %w", submission, err)
	}
	var ids []grants.GrantID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to read a grant held for %s: %w", submission, err)
		}
		ids = append(ids, grants.GrantID(id))
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("failed to list the grants held for %s: %w", submission, err)
	}
	held := make([]grants.Grant, 0, len(ids))
	for _, id := range ids {
		grant, err := s.Grant(ctx, id)
		if err != nil {
			return nil, err
		}
		held = append(held, grant)
	}
	return held, nil
}

func (s *Store) Save(ctx context.Context, grant grants.Grant) error {
	kept := grant.TakeSnapshot()
	tx := s.db.On(ctx)
	if kept.Held {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO time_held (grant_id, submission_id, period, held_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (grant_id) DO NOTHING
		`, kept.ID.String(), uuid.UUID(kept.Submission), string(kept.Period), kept.HeldAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep the time held by %s: %w", kept.ID, err)
		}
	}
	if kept.Granted {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO time_grants (id, flag, added, granted_at) VALUES ($1, $2, make_interval(secs => $3), $4)
			ON CONFLICT (id) DO NOTHING
		`, kept.ID.String(), kept.Flag.String(), kept.Added.Seconds(), kept.GrantedAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep the grant %s: %w", kept.ID, err)
		}
	}
	if kept.TakenBack {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO time_takebacks (grant_id, decided_at) VALUES ($1, $2) ON CONFLICT (grant_id) DO NOTHING
		`, kept.ID.String(), kept.TakenBackAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep that %s was taken back: %w", kept.ID, err)
		}
	}
	return nil
}
