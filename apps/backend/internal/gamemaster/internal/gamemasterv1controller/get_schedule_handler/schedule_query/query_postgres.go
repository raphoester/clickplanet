package schedule_query

import (
	"context"
	"fmt"
	"time"

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

// Schedule is every entry, the past ones too, and what plays now by schedule.Schedule's rule.
func (q *PostgresQuery) Schedule(ctx context.Context) (*gamemasterv1.GetScheduleResponse, error) {
	now := q.clock.Now()
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, rule_set, kind, description, starts_at, ends_at, priority, made_at FROM schedule_entries
		ORDER BY starts_at, made_at, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &gamemasterv1.GetScheduleResponse{}
	var entries []schedule.Entry
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
		wireKind, err := schedulemessage.KindOf(entry.Kind)
		if err != nil {
			return nil, fmt.Errorf("failed to read the entry %s: %w", entry.ID, err)
		}
		entries = append(entries, entry.Restore())
		answer.Entries = append(answer.Entries, &gamemasterv1.Entry{
			Id: entry.ID.String(), RuleSet: set, StartsAt: timestamppb.New(entry.StartsAt),
			EndsAt: timestamppb.New(entry.EndsAt), Priority: uint32(entry.Priority), MadeAt: timestamppb.New(entry.MadeAt),
			Kind: wireKind, Description: description,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}

	sched := schedule.New(entries...)
	turn := sched.TurnAt(now).TakeSnapshot()
	playingKind, err := schedulemessage.KindOf(turn.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to read what plays: %w", err)
	}
	answer.Playing = &gamemasterv1.Playing{RuleSet: string(turn.Set), Since: stamp(sched.SinceAt(now)), Kind: playingKind}
	if turn.Entry != (schedule.EntryID{}) {
		answer.Playing.EntryId = turn.Entry.String()
	}
	return answer, nil
}

// The zero time is none: default has played since before any entry.
func stamp(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
