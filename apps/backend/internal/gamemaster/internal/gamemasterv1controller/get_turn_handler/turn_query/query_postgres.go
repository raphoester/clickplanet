package turn_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// Turn is what plays now by schedule.Schedule's rule, in the players' words: no set's name.
func (q *PostgresQuery) Turn(ctx context.Context) (*gamemasterv1.Turn, error) {
	now := q.clock.Now()
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, rule_set, kind, description, starts_at, ends_at, priority, made_at FROM schedule_entries`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		entries []schedule.Entry
		ends    = map[schedule.EntryID]*timestamppb.Timestamp{}
	)
	for rows.Next() {
		var (
			id                     uuid.UUID
			set, kind, description string
			priority               int64
			entry                  schedule.EntrySnapshot
		)
		if err := rows.Scan(&id, &set, &kind, &description, &entry.StartsAt, &entry.EndsAt, &priority, &entry.MadeAt); err != nil {
			return nil, fmt.Errorf("failed to read an entry of the schedule: %w", err)
		}
		entry.ID, entry.Set, entry.Kind = schedule.EntryID(id), schedule.SetName(set), schedule.Kind(kind)
		entry.Description, entry.Priority = schedule.Description(description), schedule.Priority(priority) //nolint:gosec // CHECK (priority >= 0 AND priority <= 4294967295).
		entries = append(entries, entry.Restore())
		ends[entry.ID] = timestamppb.New(entry.EndsAt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}

	sched := schedule.New(entries...)
	turn := sched.TurnAt(now).TakeSnapshot()
	kind, err := schedulemessage.KindOf(turn.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to read what plays: %w", err)
	}
	answer := &gamemasterv1.Turn{Kind: kind, EndsAt: ends[turn.Entry], Description: string(turn.Description)}
	if _, runs := ends[turn.Entry]; runs {
		answer.EntryId = turn.Entry.String()
	}
	if since := sched.SinceAt(now); !since.IsZero() {
		answer.Since = timestamppb.New(since)
	}
	return answer, nil
}
