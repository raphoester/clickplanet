package postgres_cast_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Keep(ctx context.Context, cast casts.Cast) error {
	kept := cast.TakeSnapshot()

	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO casts (cast_id, made_at, since, until) VALUES ($1, $2, $3, $4)
		`, uuid.UUID(kept.ID), kept.MadeAt, kept.Window.Since, kept.Window.Until); err != nil {
			return fmt.Errorf("failed to keep the cast: %w", err)
		}
		for _, fighter := range kept.Fighters {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO cast_fighters (cast_id, number, account, name) VALUES ($1, $2, $3, $4)
			`, uuid.UUID(kept.ID), int64(fighter.Number), uuid.UUID(fighter.Account), fighter.Name); err != nil {
				return fmt.Errorf("failed to keep fighter %d of the cast: %w", fighter.Number, err)
			}
		}
		return nil
	})
}

func (s *Store) Cast(ctx context.Context, id casts.CastID) (casts.Cast, error) {
	snapshot := casts.CastSnapshot{ID: id}
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT made_at, since, until FROM casts WHERE cast_id = $1`, uuid.UUID(id)).
		Scan(&snapshot.MadeAt, &snapshot.Window.Since, &snapshot.Window.Until)
	if errors.Is(err, sql.ErrNoRows) {
		return casts.Cast{}, casts.ErrNoCast
	}
	if err != nil {
		return casts.Cast{}, fmt.Errorf("failed to read the cast: %w", err)
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT number, account, COALESCE(name, '') FROM cast_fighters WHERE cast_id = $1 ORDER BY number
	`, uuid.UUID(id))
	if err != nil {
		return casts.Cast{}, fmt.Errorf("failed to read the fighters of the cast: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			fighter casts.FighterSnapshot
			number  int64
			account uuid.NullUUID
		)
		if err := rows.Scan(&number, &account, &fighter.Name); err != nil {
			return casts.Cast{}, fmt.Errorf("failed to read a fighter of the cast: %w", err)
		}
		fighter.Number = casts.Number(number) //nolint:gosec // the CHECK keeps it a positive number of a cast.
		if account.Valid {
			fighter.Account = casts.AccountID(account.UUID)
		}
		snapshot.Fighters = append(snapshot.Fighters, fighter)
	}
	if err := rows.Err(); err != nil {
		return casts.Cast{}, fmt.Errorf("failed to read the fighters of the cast: %w", err)
	}
	return snapshot.Restore(), nil
}

func (s *Store) MadeBefore(ctx context.Context, at time.Time) ([]casts.CastID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT cast_id FROM casts WHERE made_at < $1`, at)
	if err != nil {
		return nil, fmt.Errorf("failed to find the casts made before %s: %w", at.Format(time.RFC3339), err)
	}
	defer func() { _ = rows.Close() }()

	var ids []casts.CastID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to read a cast's id: %w", err)
		}
		ids = append(ids, casts.CastID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to find the casts made before %s: %w", at.Format(time.RFC3339), err)
	}
	return ids, nil
}

func (s *Store) Forget(ctx context.Context, id casts.CastID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM casts WHERE cast_id = $1`, uuid.UUID(id)); err != nil {
		return fmt.Errorf("failed to forget the cast: %w", err)
	}
	return nil
}

// Its numbers stay the casts', naming no one.
func (s *Store) ForgetAccount(ctx context.Context, account casts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		UPDATE cast_fighters SET account = NULL, name = NULL WHERE account = $1
	`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account in the casts: %w", err)
	}
	return nil
}
