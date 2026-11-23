package postgres_flag_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ catalog.Store = (*Store)(nil)

func (s *Store) Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error) {
	row := s.db.On(ctx).QueryRowContext(ctx,
		`SELECT id, name, state::text, decided_at FROM flags WHERE id = $1`, id.String())
	flag, err := flagOf(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Flag{}, fmt.Errorf("%s: %w", id, catalog.ErrUnknownFlag)
	}
	return flag, err
}

func (s *Store) Flags(ctx context.Context) ([]catalog.Flag, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT id, name, state::text, decided_at FROM flags ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var flags []catalog.Flag
	for rows.Next() {
		flag, err := flagOf(rows.Scan)
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the flags: %w", err)
	}
	return flags, nil
}

func flagOf(scan func(dest ...any) error) (catalog.Flag, error) {
	var id, name, state string
	var kept catalog.Snapshot
	if err := scan(&id, &name, &state, &kept.DecidedAt); err != nil {
		return catalog.Flag{}, fmt.Errorf("failed to read a flag: %w", err)
	}
	flagID, err := catalog.ParseFlagID(id)
	if err != nil {
		return catalog.Flag{}, err //nolint:wrapcheck // the parse names it.
	}
	flagName, err := catalog.ParseName(name)
	if err != nil {
		return catalog.Flag{}, fmt.Errorf("the flag %s: %w", flagID, err)
	}
	kept.ID, kept.Name, kept.State, kept.DecidedAt = flagID, flagName, catalog.State(state), kept.DecidedAt.UTC()
	flag, err := kept.Restore()
	if err != nil {
		return catalog.Flag{}, fmt.Errorf("failed to restore a flag kept: %w", err)
	}
	return flag, nil
}

func (s *Store) Save(ctx context.Context, flags ...catalog.Flag) error {
	for _, flag := range flags {
		kept := flag.TakeSnapshot()
		if _, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO flags (id, name, state, decided_at) VALUES ($1, $2, $3::flag_state, $4)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, state = EXCLUDED.state, decided_at = EXCLUDED.decided_at
		`, kept.ID.String(), kept.Name.String(), string(kept.State), kept.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep the flag %s: %w", kept.ID, err)
		}
	}
	return nil
}
