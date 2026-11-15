package postgres_mission_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
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

func (s *Store) Brief(ctx context.Context, day days.Day, account missions.AccountID) (missions.Brief, error) {
	var kept []byte
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT missions FROM mission_briefs WHERE day = $1 AND account_id = $2
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
	ctx context.Context, day days.Day, account missions.AccountID, brief missions.Brief,
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
		INSERT INTO mission_briefs (day, account_id, missions) VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (day, account_id) DO NOTHING
	`, day.String(), uuid.UUID(account), encoded); err != nil {
		return missions.Brief{}, fmt.Errorf("failed to keep the day's brief: %w", err)
	}
	return s.Brief(ctx, day, account)
}

func (s *Store) Sheet(ctx context.Context, day days.Day, account missions.AccountID) (missions.Sheet, error) {
	return sheetOf(ctx, s.db.On(ctx), day, account)
}

func sheetOf(ctx context.Context, db cppg.Querier, day days.Day, account missions.AccountID) (missions.Sheet, error) {
	snapshot := missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{}, Paid: cpcolls.NewSet[missions.MissionID]()}

	var (
		takes, bombs, shields, fortifies, takesIndex int64
		boxes, answers                               sql.NullTime
	)
	err := db.QueryRowContext(ctx, `
		SELECT `+countedColumns+` FROM mission_sheets WHERE day = $1 AND account_id = $2
	`, day.String(), uuid.UUID(account)).Scan(&takes, &bombs, &shields, &fortifies, &boxes, &answers, &takesIndex)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's last facts: %w", err)
	}
	snapshot.Counted = missions.Counted{
		TakesBelow:     missions.LedgerMark{Position: uint64(takes), Index: uint32(takesIndex)}, //nolint:gosec // CHECK (>= 0).
		BombsBelow:     uint64(bombs),                                                           //nolint:gosec // as above.
		ShieldsBelow:   uint64(shields),                                                         //nolint:gosec // as above.
		FortifiesBelow: uint64(fortifies),                                                       //nolint:gosec // as above.
		BoxesUntil:     timeOf(boxes),
		AnswersUntil:   timeOf(answers),
	}

	if err := perMission(ctx, db, `SELECT mission_id, progress FROM mission_progress WHERE day = $1 AND account_id = $2`,
		day, account, func(id missions.MissionID, progress uint32) { snapshot.Progress[id] = progress }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's progress: %w", err)
	}
	if err := perMission(ctx, db, `SELECT mission_id, 1 FROM mission_payments WHERE day = $1 AND account_id = $2`,
		day, account, func(id missions.MissionID, _ uint32) { snapshot.Paid.Add(id) }); err != nil {
		return missions.Sheet{}, fmt.Errorf("failed to read the day's payments: %w", err)
	}

	return missions.SheetOf(snapshot), nil
}

func perMission(
	ctx context.Context, db cppg.Querier, query string, day days.Day, account missions.AccountID,
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
	ctx context.Context, day days.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet,
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

		if counted := after.Counted; counted != before.Counted {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mission_sheets (day, account_id, `+countedColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (day, account_id) DO UPDATE SET
					counted_takes_below = excluded.counted_takes_below,
					counted_bombs_below = excluded.counted_bombs_below,
					counted_shields_below = excluded.counted_shields_below,
					counted_fortifies_below = excluded.counted_fortifies_below,
					counted_boxes_until = excluded.counted_boxes_until,
					counted_answers_until = excluded.counted_answers_until,
					counted_takes_below_index = excluded.counted_takes_below_index
			`, day.String(), uuid.UUID(account),
				int64(counted.TakesBelow.Position), int64(counted.BombsBelow), //nolint:gosec // ledger positions fit in int64.
				int64(counted.ShieldsBelow), int64(counted.FortifiesBelow), //nolint:gosec // as above.
				nullableTime(counted.BoxesUntil), nullableTime(counted.AnswersUntil), int64(counted.TakesBelow.Index)); err != nil {
				return fmt.Errorf("failed to keep the day's last facts: %w", err)
			}
		}
		for id, progress := range after.Progress {
			if progress == before.Progress[id] {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO mission_progress (day, account_id, mission_id, progress) VALUES ($1, $2, $3, $4)
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
				INSERT INTO mission_payments (day, account_id, mission_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
			`, day.String(), uuid.UUID(account), uuid.UUID(id)); err != nil {
				return fmt.Errorf("failed to keep the payment of mission %s: %w", id, err)
			}
		}
		return nil
	})
}

func (s *Store) PaidTotals(ctx context.Context, account missions.AccountID) (missions.Paid, error) {
	paid, err := paidTo(ctx, s.db.On(ctx), []missions.AccountID{account})
	if err != nil {
		return missions.Paid{}, err
	}
	return paid[account], nil
}

func (s *Store) PaidAfter(ctx context.Context, after missions.AccountID, limit int) ([]missions.AccountPaid, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT DISTINCT account_id FROM mission_payments WHERE account_id > $1 ORDER BY account_id LIMIT $2
	`, uuid.UUID(after), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read a page of the accounts paid: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var accounts []missions.AccountID
	for rows.Next() {
		var account uuid.UUID
		if err := rows.Scan(&account); err != nil {
			return nil, fmt.Errorf("failed to read an account paid: %w", err)
		}
		accounts = append(accounts, missions.AccountID(account))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read a page of the accounts paid: %w", err)
	}

	paid, err := paidTo(ctx, s.db.On(ctx), accounts)
	if err != nil {
		return nil, err
	}
	page := make([]missions.AccountPaid, 0, len(accounts))
	for _, account := range accounts {
		page = append(page, missions.AccountPaid{Account: account, Paid: paid[account]})
	}
	return page, nil
}

// Each account's payments by the kind of the mission in the day's brief, with its goals summed. A payment whose brief
// holds no such mission has no kind: it counts in all only.
const paidByKind = `
	SELECT pay.account_id, mission.kind, count(*), COALESCE(sum(mission.goal), 0)
	FROM mission_payments AS pay
	LEFT JOIN mission_briefs AS brief ON brief.day = pay.day AND brief.account_id = pay.account_id
	LEFT JOIN LATERAL (
		SELECT m->>'kind' AS kind, (m->'payload'->>'goal')::bigint AS goal
		FROM jsonb_array_elements(brief.missions) AS m WHERE (m->>'id')::uuid = pay.mission_id
	) AS mission ON true
	WHERE pay.account_id = ANY($1::uuid[])
	GROUP BY pay.account_id, mission.kind`

func paidTo(ctx context.Context, db cppg.Querier, accounts []missions.AccountID) (map[missions.AccountID]missions.Paid, error) {
	paid := make(map[missions.AccountID]missions.Paid, len(accounts))
	if len(accounts) == 0 {
		return paid, nil
	}
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
		paid[account] = missions.NothingPaid()
	}

	rows, err := db.QueryContext(ctx, paidByKind, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to count the accounts' payments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account      uuid.UUID
			name         sql.NullString
			count, goals int64
		)
		if err := rows.Scan(&account, &name, &count, &goals); err != nil {
			return nil, fmt.Errorf("failed to count an account's payments: %w", err)
		}
		totals := paid[missions.AccountID(account)]
		totals.InAll += uint64(count) //nolint:gosec // a count is never negative.
		if name.Valid {
			kind, ok := kindOfRecord[name.String]
			if !ok {
				return nil, fmt.Errorf("%w: %q", ErrUnknownKind, name.String)
			}
			totals.ByKind[kind] = missions.KindPaid{Done: uint64(count), Goals: uint64(goals)} //nolint:gosec // as above, and goals are never negative.
		}
		paid[missions.AccountID(account)] = totals
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to count the accounts' payments: %w", err)
	}
	return paid, nil
}

const countedColumns = `counted_takes_below, counted_bombs_below, counted_shields_below, counted_fortifies_below,
	counted_boxes_until, counted_answers_until, counted_takes_below_index`

func timeOf(at sql.NullTime) time.Time {
	if !at.Valid {
		return time.Time{}
	}
	return at.Time.UTC()
}

func nullableTime(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
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
