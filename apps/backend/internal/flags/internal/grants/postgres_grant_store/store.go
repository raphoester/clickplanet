package postgres_grant_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
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
	var flag string
	var addedMicros int64
	err := tx.QueryRowContext(ctx, `
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
	if !kept.Granted && !kept.TakenBack {
		return grants.Unheard(id), nil
	}
	grant, err := kept.Restore()
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to restore a grant kept: %w", err)
	}
	return grant, nil
}

func (s *Store) Save(ctx context.Context, grant grants.Grant) error {
	kept := grant.TakeSnapshot()
	tx := s.db.On(ctx)
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
