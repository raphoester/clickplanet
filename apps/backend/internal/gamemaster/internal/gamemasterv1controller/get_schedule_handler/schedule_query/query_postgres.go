package schedule_query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
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

// Schedule is every entry, the past ones too, every season's finale, and what plays now by schedule.Schedule's rule.
func (q *PostgresQuery) Schedule(ctx context.Context) (*gamemasterv1.GetScheduleResponse, error) {
	now := q.clock.Now()
	calendar, err := q.calendar(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, rule_set, kind, description, starts_at, ends_at, priority, times, made_at FROM schedule_entries
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
			priority, times        int64
			entry                  schedule.EntrySnapshot
		)
		if err := rows.Scan(&id, &set, &kind, &description, &entry.StartsAt, &entry.EndsAt, &priority, &times, &entry.MadeAt); err != nil {
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
			Kind: wireKind, Description: description, Times: uint32(times), //nolint:gosec // CHECK times BETWEEN 0 AND 10.
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}

	sched := schedule.New(entries...).With(calendar.Finales()...)
	turn := sched.TurnAt(now).TakeSnapshot()
	playingKind, err := schedulemessage.KindOf(turn.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to read what plays: %w", err)
	}
	answer.Playing = &gamemasterv1.Playing{RuleSet: string(turn.Set), Since: stamp(sched.SinceAt(now)), Kind: playingKind}
	if turn.Entry != (schedule.EntryID{}) {
		answer.Playing.EntryId = turn.Entry.String()
	}
	if running, ok := calendar.RunningAt(now); ok && turn.Finale {
		number := uint32(running.Number)
		answer.Playing.FinaleSeason = &number
	}
	for _, window := range calendar.Windows() {
		if window.Finale.Planned() {
			answer.Finales = append(answer.Finales, seasonmessage.ScheduledFinaleOf(window))
		}
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

func (q *PostgresQuery) calendar(ctx context.Context) (season.Calendar, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT s.number, (EXTRACT(EPOCH FROM s.break_before) * 1000000)::bigint, (EXTRACT(EPOCH FROM s.length) * 1000000)::bigint,
			(EXTRACT(EPOCH FROM f.length) * 1000000)::bigint, f.rule_set, f.description, f.times
		FROM seasons AS s LEFT JOIN season_finales AS f ON f.season = s.number`)
	if err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var seasons []season.Season
	for rows.Next() {
		var (
			number, brk, length       int64
			finaleLength, times       sql.NullInt64
			finaleSet, finaleDescribe sql.NullString
		)
		if err := rows.Scan(&number, &brk, &length, &finaleLength, &finaleSet, &finaleDescribe, &times); err != nil {
			return season.Calendar{}, fmt.Errorf("failed to read a season: %w", err)
		}
		planned := season.Snapshot{
			Number: season.Number(number), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
			Break:  time.Duration(brk) * time.Microsecond, Length: time.Duration(length) * time.Microsecond,
		}
		if finaleLength.Valid {
			planned.Finale = season.Finale{
				Length: time.Duration(finaleLength.Int64) * time.Microsecond, Set: schedule.SetName(finaleSet.String),
				Description: schedule.Description(finaleDescribe.String),
				Times:       schedule.Times(times.Int64), //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
			}
		}
		seasons = append(seasons, planned.Restore())
	}
	if err := rows.Err(); err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return season.NewCalendar(seasons...), nil
}
