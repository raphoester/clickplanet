package postgres_flag_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ catalog.Store = (*Store)(nil)

const columns = `f.id, f.name, f.state::text, f.decided_at, f.ends_at,
	f.picture_hash, f.picture_width, f.picture_height, f.picture_stretch, f.picture_focus_x, f.picture_focus_y,
	f.pin_latitude, f.pin_longitude, COALESCE(c.note, '')`

const from = ` FROM flags f LEFT JOIN closures c ON c.flag = f.id`

const (
	uniqueViolation = "23505"
	uniqueNameIndex = "flags_name_key"
)

func (s *Store) Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error) {
	// Locked to the end of the transaction a change reads it in, so two changes never undo each other.
	row := s.db.On(ctx).QueryRowContext(ctx, `SELECT `+columns+from+` WHERE f.id = $1 FOR UPDATE OF f`, id.String())
	flag, err := flagOf(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Flag{}, fmt.Errorf("%s: %w", id, catalog.ErrUnknownFlag)
	}
	return flag, err
}

func (s *Store) Flags(ctx context.Context) ([]catalog.Flag, error) {
	return s.read(ctx, `SELECT `+columns+from+` ORDER BY f.id`)
}

func (s *Store) Ending(ctx context.Context, at time.Time) ([]catalog.Flag, error) {
	return s.read(ctx, `SELECT `+columns+from+` WHERE f.state = 'open' AND f.ends_at <= $1 ORDER BY f.ends_at, f.id`, at.UTC())
}

func (s *Store) read(ctx context.Context, query string, args ...any) ([]catalog.Flag, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, query, args...)
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
	var id, name, state, note string
	var kept catalog.Snapshot
	var picture pictures.Snapshot
	var focusX, focusY float32
	if err := scan(&id, &name, &state, &kept.DecidedAt, &kept.EndsAt,
		&picture.Hash, &picture.Width, &picture.Height, &picture.Fit.Stretch, &focusX, &focusY,
		&kept.Pin.Latitude, &kept.Pin.Longitude, &note); err != nil {
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
	picture.Fit.FocusX, picture.Fit.FocusY = float64(focusX), float64(focusY)
	kept.ID, kept.Name, kept.State, kept.Picture, kept.Note = flagID, flagName, catalog.State(state), picture,
		catalog.Note(note)
	kept.DecidedAt, kept.EndsAt = kept.DecidedAt.UTC(), kept.EndsAt.UTC()
	flag, err := kept.Restore()
	if err != nil {
		return catalog.Flag{}, fmt.Errorf("failed to restore a flag kept: %w", err)
	}
	return flag, nil
}

func (s *Store) Save(ctx context.Context, flags ...catalog.Flag) error {
	for _, flag := range flags {
		kept := flag.TakeSnapshot()
		picture := kept.Picture
		_, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO flags (id, name, state, decided_at, ends_at,
				picture_hash, picture_width, picture_height, picture_stretch, picture_focus_x, picture_focus_y,
				pin_latitude, pin_longitude)
			VALUES ($1, $2, $3::flag_state, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, state = EXCLUDED.state,
				decided_at = EXCLUDED.decided_at, ends_at = EXCLUDED.ends_at, picture_hash = EXCLUDED.picture_hash,
				picture_width = EXCLUDED.picture_width, picture_height = EXCLUDED.picture_height,
				picture_stretch = EXCLUDED.picture_stretch, picture_focus_x = EXCLUDED.picture_focus_x,
				picture_focus_y = EXCLUDED.picture_focus_y, pin_latitude = EXCLUDED.pin_latitude,
				pin_longitude = EXCLUDED.pin_longitude
		`, kept.ID.String(), kept.Name.String(), string(kept.State), kept.DecidedAt.UTC(), kept.EndsAt.UTC(),
			picture.Hash, picture.Width, picture.Height, picture.Fit.Stretch, float32(picture.Fit.FocusX),
			float32(picture.Fit.FocusY), kept.Pin.Latitude, kept.Pin.Longitude)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation && pqErr.Constraint == uniqueNameIndex {
			return fmt.Errorf("%s: %w", kept.Name, catalog.ErrNameTaken)
		}
		if err != nil {
			return fmt.Errorf("failed to keep the flag %s: %w", kept.ID, err)
		}
		if err := s.keepNote(ctx, kept); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) keepNote(ctx context.Context, kept catalog.Snapshot) error {
	var err error
	if kept.Note == "" {
		_, err = s.db.On(ctx).ExecContext(ctx, `DELETE FROM closures WHERE flag = $1`, kept.ID.String())
	} else {
		_, err = s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO closures (flag, note) VALUES ($1, $2)
			ON CONFLICT (flag) DO UPDATE SET note = EXCLUDED.note
		`, kept.ID.String(), kept.Note.String())
	}
	if err != nil {
		return fmt.Errorf("failed to keep the close's note of %s: %w", kept.ID, err)
	}
	return nil
}
