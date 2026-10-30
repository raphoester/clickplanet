package postgres_flag_day_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ flagmissions.Store = (*Store)(nil)

const selectDay = `
	SELECT day.takes, day.counted_takes_below, day.counted_takes_below_index, day.strikes, day.counted_strikes_below,
	       day.counted_strikes_below_index,
	       mission.mission_id, mission.kind::text, mission.payload, mission.points, mission.progress, mission.done
	FROM flag_days AS day LEFT JOIN flag_missions AS mission USING (day, flag)
	WHERE day.day = $1 AND day.flag = $2`

func (s *Store) Day(ctx context.Context, day days.Day, flag string) (flagmissions.Day, error) {
	return dayOf(s.db.On(ctx).QueryRowContext(ctx, selectDay, day.String(), flag))
}

func dayOf(row *sql.Row) (flagmissions.Day, error) {
	var (
		snapshot              flagmissions.DaySnapshot
		takes, below, strikes int64
		belowIndex, struck    int64
		struckIndex           int64
		id                    uuid.NullUUID
		kind                  sql.NullString
		payload               []byte
		points, progress      sql.NullInt64
		done                  sql.NullBool
	)
	err := row.Scan(&takes, &below, &belowIndex, &strikes, &struck, &struckIndex, &id, &kind, &payload, &points, &progress, &done)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return flagmissions.Day{}, flagmissions.ErrNoDay
	case err != nil:
		return flagmissions.Day{}, fmt.Errorf("failed to read the flag's day: %w", err)
	}
	//nolint:gosec // CHECK (... >= 0) on each.
	snapshot.Takes, snapshot.Strikes = uint64(takes), uint64(strikes)
	snapshot.TakesBelow = missions.LedgerMark{Position: uint64(below), Index: uint32(belowIndex)}     //nolint:gosec // as above.
	snapshot.StrikesBelow = missions.LedgerMark{Position: uint64(struck), Index: uint32(struckIndex)} //nolint:gosec // as above.

	if id.Valid {
		order, err := orderOf(kind.String, payload)
		if err != nil {
			return flagmissions.Day{}, err
		}
		//nolint:gosec // CHECK (points > 0), CHECK (progress >= 0).
		snapshot.Mission = &flagmissions.MissionSnapshot{ID: flagmissions.MissionID(id.UUID), Order: order, Points: uint32(points.Int64)}
		snapshot.Progress, snapshot.Done = uint32(progress.Int64), done.Bool //nolint:gosec // CHECK (progress >= 0).
	}
	return flagmissions.DayOf(snapshot) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepDay(
	ctx context.Context, day days.Day, flag string, drawn flagmissions.Day,
) (flagmissions.Day, error) {
	snapshot := drawn.Snapshot()
	if err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		kept, err := tx.ExecContext(ctx, `
			INSERT INTO flag_days (day, flag, takes, counted_takes_below, counted_takes_below_index, strikes,
				counted_strikes_below, counted_strikes_below_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (day, flag) DO NOTHING
		`, day.String(), flag, int64(snapshot.Takes), int64(snapshot.TakesBelow.Position), //nolint:gosec // counts fit in int64.
			int64(snapshot.TakesBelow.Index), int64(snapshot.Strikes), int64(snapshot.StrikesBelow.Position), //nolint:gosec // as above.
			int64(snapshot.StrikesBelow.Index))
		if err != nil {
			return fmt.Errorf("failed to keep %s's day: %w", flag, err)
		}
		first, err := kept.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to keep %s's day: %w", flag, err)
		}
		if first == 0 || snapshot.Mission == nil {
			return nil
		}

		kind, payload, err := recordOf(snapshot.Mission.Order)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO flag_missions (day, flag, mission_id, kind, payload, points, progress, done)
			VALUES ($1, $2, $3, $4::flag_mission_kind, $5::jsonb, $6, $7, $8)
		`, day.String(), flag, uuid.UUID(snapshot.Mission.ID), kind, payload, int64(snapshot.Mission.Points),
			int64(snapshot.Progress), snapshot.Done); err != nil {
			return fmt.Errorf("failed to keep %s's mission: %w", flag, err)
		}
		return nil
	}); err != nil {
		return flagmissions.Day{}, err //nolint:wrapcheck // named inside.
	}
	return s.Day(ctx, day, flag)
}

func (s *Store) ChangeDay(
	ctx context.Context, day days.Day, flag string, change func(flagmissions.Day) flagmissions.Day,
) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		current, err := dayOf(tx.QueryRowContext(ctx, selectDay+` FOR UPDATE OF day`, day.String(), flag))
		if err != nil {
			return err
		}
		before, after := current.Snapshot(), change(current).Snapshot()

		if after.Takes != before.Takes || after.TakesBelow != before.TakesBelow || after.Strikes != before.Strikes ||
			after.StrikesBelow != before.StrikesBelow {
			if _, err := tx.ExecContext(ctx, `
				UPDATE flag_days SET takes = $3, counted_takes_below = $4, counted_takes_below_index = $5, strikes = $6,
					counted_strikes_below = $7, counted_strikes_below_index = $8
				WHERE day = $1 AND flag = $2
			`, day.String(), flag, int64(after.Takes), int64(after.TakesBelow.Position), //nolint:gosec // counts fit in int64.
				int64(after.TakesBelow.Index), int64(after.Strikes), int64(after.StrikesBelow.Position), //nolint:gosec // as above.
				int64(after.StrikesBelow.Index)); err != nil {
				return fmt.Errorf("failed to keep what %s did: %w", flag, err)
			}
		}
		if after.Mission != nil && (after.Progress != before.Progress || after.Done != before.Done) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE flag_missions SET progress = $3, done = $4 WHERE day = $1 AND flag = $2
			`, day.String(), flag, int64(after.Progress), after.Done); err != nil {
				return fmt.Errorf("failed to keep how far %s took its mission: %w", flag, err)
			}
		}
		return nil
	})
}
