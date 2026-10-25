package postgres_mission_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ missions.Store = (*Store)(nil)

const sheetLock = 0x6d697373

func (s *Store) Brief(ctx context.Context, day missions.Day, flag string) (missions.Brief, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT slot, kind, target, goal, reward, reward_amount
		FROM mission_briefs
		WHERE day = $1::date AND flag = $2
		ORDER BY slot
	`, day.String(), flag)
	if err != nil {
		return missions.Brief{}, fmt.Errorf("failed to read %s's brief: %w", flag, err)
	}
	defer func() { _ = rows.Close() }()

	var snapshots []missions.MissionSnapshot
	for rows.Next() {
		var snapshot missions.MissionSnapshot
		if err := rows.Scan(&snapshot.Slot, &snapshot.Kind, &snapshot.Target, &snapshot.Goal,
			&snapshot.Reward.Kind, &snapshot.Reward.Amount); err != nil {
			return missions.Brief{}, fmt.Errorf("failed to scan a mission of %s's brief: %w", flag, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return missions.Brief{}, fmt.Errorf("failed to read %s's brief: %w", flag, err)
	}
	if len(snapshots) == 0 {
		return missions.Brief{}, missions.ErrNoBrief
	}
	return missions.BriefOf(snapshots) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepBrief(ctx context.Context, day missions.Day, flag string, brief missions.Brief) (missions.Brief, error) {
	for _, mission := range brief.Snapshot() {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO mission_briefs (day, flag, slot, kind, target, goal, reward, reward_amount)
			VALUES ($1::date, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (day, flag, slot) DO NOTHING
		`, day.String(), flag, int16(mission.Slot), string(mission.Kind), mission.Target, int64(mission.Goal),
			string(mission.Reward.Kind), int64(mission.Reward.Amount)); err != nil {
			return missions.Brief{}, fmt.Errorf("failed to keep %s's brief: %w", flag, err)
		}
	}
	return s.Brief(ctx, day, flag)
}

func (s *Store) Sheet(ctx context.Context, day missions.Day, account players.AccountID) (missions.Sheet, error) {
	return sheetOf(ctx, s.db, day, account)
}

func sheetOf(ctx context.Context, db cppg.Querier, day missions.Day, account players.AccountID) (missions.Sheet, error) {
	snapshot := missions.SheetSnapshot{Taken: map[missions.Slot]uint32{}, Collected: cpcolls.NewSet[missions.Slot]()}

	if err := db.QueryRowContext(ctx, `
		SELECT coalesce((SELECT flag FROM mission_sheets WHERE day = $1::date AND account_id = $2), '')
	`, day.String(), uuid.UUID(account)).Scan(&snapshot.Flag); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's flag: %w", err)
	}

	if err := slots(ctx, db, `SELECT slot, taken FROM mission_progress WHERE day = $1::date AND account_id = $2`,
		day, account, func(slot missions.Slot, taken uint32) { snapshot.Taken[slot] = taken }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's progress: %w", err)
	}
	if err := slots(ctx, db, `SELECT slot, 1 FROM mission_collections WHERE day = $1::date AND account_id = $2`,
		day, account, func(slot missions.Slot, _ uint32) { snapshot.Collected.Add(slot) }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's collections: %w", err)
	}

	return missions.SheetOf(snapshot), nil
}

func slots(
	ctx context.Context, db cppg.Querier, query string, day missions.Day, account players.AccountID,
	see func(missions.Slot, uint32),
) error {
	rows, err := db.QueryContext(ctx, query, day.String(), uuid.UUID(account))
	if err != nil {
		return err //nolint:wrapcheck // the caller says what was read.
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			slot  missions.Slot
			count uint32
		)
		if err := rows.Scan(&slot, &count); err != nil {
			return err //nolint:wrapcheck // the caller says what was read.
		}
		see(slot, count)
	}
	return rows.Err() //nolint:wrapcheck // the caller says what was read.
}

func (s *Store) ChangeSheet(
	ctx context.Context, day missions.Day, account players.AccountID, change func(missions.Sheet) missions.Sheet,
) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin changing the day's missions: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, sheetLock, account.String()); err != nil {
		return fmt.Errorf("failed to lock the account's missions: %w", err)
	}

	current, err := sheetOf(ctx, tx, day, account)
	if err != nil {
		return err
	}
	before, after := current.Snapshot(), change(current).Snapshot()

	if after.Flag != "" && after.Flag != before.Flag {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO mission_sheets (day, account_id, flag) VALUES ($1::date, $2, $3)
			ON CONFLICT (day, account_id) DO UPDATE SET flag = excluded.flag
		`, day.String(), uuid.UUID(account), after.Flag); err != nil {
			return fmt.Errorf("failed to keep the day's flag: %w", err)
		}
	}
	for slot, taken := range after.Taken {
		if taken == before.Taken[slot] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO mission_progress (day, account_id, slot, taken) VALUES ($1::date, $2, $3, $4)
			ON CONFLICT (day, account_id, slot) DO UPDATE SET taken = excluded.taken
		`, day.String(), uuid.UUID(account), int16(slot), int64(taken)); err != nil {
			return fmt.Errorf("failed to keep the progress of mission %d: %w", slot, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the day's missions: %w", err)
	}
	return nil
}

func (s *Store) Collect(
	ctx context.Context, day missions.Day, account players.AccountID, slot missions.Slot, at time.Time,
	pay func(context.Context) error,
) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin collecting mission %d: %w", slot, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	kept, err := tx.ExecContext(ctx, `
		INSERT INTO mission_collections (day, account_id, slot, collected_at) VALUES ($1::date, $2, $3, $4)
		ON CONFLICT (day, account_id, slot) DO NOTHING
	`, day.String(), uuid.UUID(account), int16(slot), at)
	if err != nil {
		return fmt.Errorf("failed to keep the collection of mission %d: %w", slot, err)
	}
	rows, err := kept.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether mission %d was collected: %w", slot, err)
	}
	if rows == 0 {
		return missions.ErrCollected
	}

	if err := pay(ctx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the collection of mission %d: %w", slot, err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	for _, table := range []string{"mission_sheets", "mission_collections"} {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE account_id = $1`, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to forget the account's %s: %w", table, err)
		}
	}
	return nil
}
