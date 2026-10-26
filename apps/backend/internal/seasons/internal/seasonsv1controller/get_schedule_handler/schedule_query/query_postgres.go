package schedule_query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrUnknownStage = errors.New("the wire has no such stage")

func NewPostgresQuery(db cppg.Conn, seasons calendar.Calendar, sets schedule.PhaseSets, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, seasons: seasons, sets: sets, clock: clock}
}

type PostgresQuery struct {
	db      cppg.Conn
	seasons calendar.Calendar
	sets    schedule.PhaseSets
	clock   cptime.Clock
}

// Schedule is every phase and every entry, the past ones too, and what plays now by schedule.Schedule's rule.
func (q *PostgresQuery) Schedule(ctx context.Context) (*seasonsv1.GetScheduleResponse, error) {
	now := q.clock.Now()
	phases := q.seasons.Phases()
	sched, err := schedule.New(phases, q.sets)
	if err != nil {
		return nil, fmt.Errorf("failed to read the phases: %w", err)
	}

	answer := &seasonsv1.GetScheduleResponse{}
	for _, phase := range phases {
		wired, err := q.phase(phase)
		if err != nil {
			return nil, err
		}
		answer.Phases = append(answer.Phases, wired)
	}

	entries, err := q.entries(ctx)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		sched = sched.With(entry.Restore())
		answer.Entries = append(answer.Entries, &seasonsv1.Entry{
			Id: entry.ID.String(), RuleSet: string(entry.Set), StartsAt: timestamppb.New(entry.StartsAt),
			EndsAt: timestamppb.New(entry.EndsAt), Priority: uint32(entry.Priority), MadeAt: timestamppb.New(entry.MadeAt),
		})
	}

	turn := sched.TurnAt(now).TakeSnapshot()
	answer.Playing = &seasonsv1.Playing{RuleSet: string(turn.Set), Since: stamp(sched.SinceAt(now))}
	if turn.Entry != (schedule.EntryID{}) {
		answer.Playing.EntryId = turn.Entry.String()
	}
	return answer, nil
}

func (q *PostgresQuery) phase(phase calendar.Phase) (*seasonsv1.Phase, error) {
	set, err := q.sets.SetOf(phase.Stage)
	if err != nil {
		return nil, fmt.Errorf("failed to read the set of a phase: %w", err)
	}
	stage, err := stageOf(phase.Stage)
	if err != nil {
		return nil, err
	}
	return &seasonsv1.Phase{
		Season: uint32(phase.Season), Stage: stage, RuleSet: string(set),
		StartsAt: stamp(phase.StartsAt), EndsAt: stamp(phase.EndsAt),
	}, nil
}

func (q *PostgresQuery) entries(ctx context.Context) ([]schedule.EntrySnapshot, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, rule_set, starts_at, ends_at, priority, made_at FROM schedule_entries ORDER BY starts_at, made_at, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []schedule.EntrySnapshot
	for rows.Next() {
		var (
			id       uuid.UUID
			set      string
			priority int64
			entry    schedule.EntrySnapshot
		)
		if err := rows.Scan(&id, &set, &entry.StartsAt, &entry.EndsAt, &priority, &entry.MadeAt); err != nil {
			return nil, fmt.Errorf("failed to read an entry of the schedule: %w", err)
		}
		entry.ID, entry.Set, entry.Priority = schedule.EntryID(id), schedule.SetName(set), schedule.Priority(priority) //nolint:gosec // CHECK (priority > 0 AND priority <= 4294967295).
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	return entries, nil
}

func stageOf(stage calendar.Stage) (seasonsv1.Stage, error) {
	switch stage {
	case calendar.StageDays:
		return seasonsv1.Stage_STAGE_DAYS, nil
	case calendar.StageFinale:
		return seasonsv1.Stage_STAGE_FINALE, nil
	case calendar.StageOver:
		return seasonsv1.Stage_STAGE_OVER, nil
	}
	return seasonsv1.Stage_STAGE_UNSPECIFIED, fmt.Errorf("%q: %w", stage, ErrUnknownStage)
}

// The zero time is none: a phase that started before anything, or never ends.
func stamp(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
