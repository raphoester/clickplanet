package postgres_mission_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ missions.Store = (*Store)(nil)

const sheetLock = 0x6d697373

func (s *Store) Brief(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Brief, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT missions FROM mission_briefs WHERE day = $1::date AND account_id = $2
	`, day.String(), uuid.UUID(account)).Scan(&kept)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return missions.Brief{}, missions.ErrNoBrief
	case err != nil:
		return missions.Brief{}, fmt.Errorf("failed to read the day's brief: %w", err)
	}

	var records []missionRecord
	if err := json.Unmarshal(kept, &records); err != nil {
		return missions.Brief{}, fmt.Errorf("failed to decode the day's brief: %w", err)
	}
	snapshots := make([]missions.MissionSnapshot, 0, len(records))
	for _, record := range records {
		snapshot, err := record.snapshot()
		if err != nil {
			return missions.Brief{}, fmt.Errorf("failed to read the day's brief: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return missions.BriefOf(snapshots) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepBrief(
	ctx context.Context, day missions.Day, account missions.AccountID, brief missions.Brief,
) (missions.Brief, error) {
	drawn := brief.Snapshot()
	records := make([]missionRecord, 0, len(drawn))
	for _, mission := range drawn {
		record, err := recordOf(mission)
		if err != nil {
			return missions.Brief{}, fmt.Errorf("failed to keep the day's brief: %w", err)
		}
		records = append(records, record)
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		return missions.Brief{}, fmt.Errorf("failed to encode the day's brief: %w", err)
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO mission_briefs (day, account_id, missions) VALUES ($1::date, $2, $3::jsonb)
		ON CONFLICT (day, account_id) DO NOTHING
	`, day.String(), uuid.UUID(account), encoded); err != nil {
		return missions.Brief{}, fmt.Errorf("failed to keep the day's brief: %w", err)
	}
	return s.Brief(ctx, day, account)
}

func (s *Store) Sheet(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Sheet, error) {
	return sheetOf(ctx, s.db.On(ctx), day, account)
}

func sheetOf(ctx context.Context, db cppg.Querier, day missions.Day, account missions.AccountID) (missions.Sheet, error) {
	snapshot := missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{}, Paid: cpcolls.NewSet[missions.MissionID]()}

	var below int64
	err := db.QueryRowContext(ctx, `
		SELECT counted_takes_below FROM mission_sheets WHERE day = $1::date AND account_id = $2
	`, day.String(), uuid.UUID(account)).Scan(&below)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's last take: %w", err)
	}
	snapshot.TakesBelow = uint64(below) //nolint:gosec // CHECK (counted_takes_below >= 0).

	if err := perMission(ctx, db, `SELECT mission_id, progress FROM mission_progress WHERE day = $1::date AND account_id = $2`,
		day, account, func(id missions.MissionID, progress uint32) { snapshot.Progress[id] = progress }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's progress: %w", err)
	}
	if err := perMission(ctx, db, `SELECT mission_id, 1 FROM mission_payments WHERE day = $1::date AND account_id = $2`,
		day, account, func(id missions.MissionID, _ uint32) { snapshot.Paid.Add(id) }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's payments: %w", err)
	}

	return missions.SheetOf(snapshot), nil
}

func perMission(
	ctx context.Context, db cppg.Querier, query string, day missions.Day, account missions.AccountID,
	see func(missions.MissionID, uint32),
) error {
	rows, err := db.QueryContext(ctx, query, day.String(), uuid.UUID(account))
	if err != nil {
		return err //nolint:wrapcheck // the caller says what was read.
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			id    uuid.UUID
			count uint32
		)
		if err := rows.Scan(&id, &count); err != nil {
			return err //nolint:wrapcheck // the caller says what was read.
		}
		see(missions.MissionID(id), count)
	}
	return rows.Err() //nolint:wrapcheck // the caller says what was read.
}

func (s *Store) ChangeSheet(
	ctx context.Context, day missions.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet,
) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, sheetLock, account.String()); err != nil {
			return fmt.Errorf("failed to lock the account's missions: %w", err)
		}

		current, err := sheetOf(ctx, tx, day, account)
		if err != nil {
			return err
		}
		before, after := current.Snapshot(), change(current).Snapshot()

		if after.TakesBelow != before.TakesBelow {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mission_sheets (day, account_id, counted_takes_below) VALUES ($1::date, $2, $3)
				ON CONFLICT (day, account_id) DO UPDATE SET counted_takes_below = excluded.counted_takes_below
			`, day.String(), uuid.UUID(account), int64(after.TakesBelow)); err != nil { //nolint:gosec // a ledger position fits in int64.
				return fmt.Errorf("failed to keep the day's last take: %w", err)
			}
		}
		for id, progress := range after.Progress {
			if progress == before.Progress[id] {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mission_progress (day, account_id, mission_id, progress) VALUES ($1::date, $2, $3, $4)
				ON CONFLICT (day, account_id, mission_id) DO UPDATE SET progress = excluded.progress
			`, day.String(), uuid.UUID(account), uuid.UUID(id), int64(progress)); err != nil {
				return fmt.Errorf("failed to keep the progress of mission %s: %w", id, err)
			}
		}
		var paid []missions.MissionID
		after.Paid.ForEach(func(id missions.MissionID) {
			if !before.Paid.Contains(id) {
				paid = append(paid, id)
			}
		})
		for _, id := range paid {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mission_payments (day, account_id, mission_id) VALUES ($1::date, $2, $3) ON CONFLICT DO NOTHING
			`, day.String(), uuid.UUID(account), uuid.UUID(id)); err != nil {
				return fmt.Errorf("failed to keep the payment of mission %s: %w", id, err)
			}
		}
		return nil
	})
}

func (s *Store) PaidInAll(ctx context.Context, account missions.AccountID) (uint64, error) {
	var paid int64
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT count(*) FROM mission_payments WHERE account_id = $1
	`, uuid.UUID(account)).Scan(&paid); err != nil {
		return 0, fmt.Errorf("failed to count the account's payments: %w", err)
	}
	return uint64(paid), nil //nolint:gosec // a count is never negative.
}

func (s *Store) DeleteAccount(ctx context.Context, account missions.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM mission_sheets WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's days: %w", err)
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM mission_briefs WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's briefs: %w", err)
	}
	return nil
}
