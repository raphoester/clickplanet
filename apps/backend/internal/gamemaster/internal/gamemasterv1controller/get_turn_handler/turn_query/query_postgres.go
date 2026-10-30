package turn_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/momentmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock, cutoff season.Cutoff) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock, cutoff: cutoff}
}

type PostgresQuery struct {
	db     cppg.Conn
	clock  cptime.Clock
	cutoff season.Cutoff
}

// Now is what plays now, the battles to come, the seasons and the day, and the last moment scoring scored, by
// schedule.Schedule's, season.Calendar's and moments.Plan's rules, in the players' words: no set's name.
func (q *PostgresQuery) Now(ctx context.Context) (*gamemasterv1.GetTurnResponse, error) {
	now := q.clock.Now()
	entries, err := q.entries(ctx)
	if err != nil {
		return nil, err
	}
	seasons, err := q.seasons(ctx)
	if err != nil {
		return nil, err
	}
	opened, err := q.opened(ctx)
	if err != nil {
		return nil, err
	}
	plan := moments.NewPlan(seasons, q.cutoff, opened, entries...)
	sched := schedule.New(entries...).With(seasons.Finales()...)

	turn, err := turnOf(sched, plan, now)
	if err != nil {
		return nil, err
	}
	timeline := timelineOf(sched.BattlesAfter(now), plan.FinalesAfter(now), plan)
	if running, ok := seasons.RunningAt(now); ok {
		timeline.Current = seasonmessage.SeasonOf(running)
	}
	if next, ok := seasons.NextAfter(now); ok {
		timeline.Next = seasonmessage.SeasonOf(next)
	}
	if today, ok := plan.DayAt(now); ok {
		timeline.Today = seasonmessage.DayOf(today)
	}
	debrief, err := q.debrief(ctx, plan)
	if err != nil {
		return nil, err
	}
	return &gamemasterv1.GetTurnResponse{Turn: turn, Timeline: timeline, Debrief: debrief}, nil
}

func turnOf(sched schedule.Schedule, plan moments.Plan, now time.Time) (*gamemasterv1.Turn, error) {
	turn := sched.TurnAt(now).TakeSnapshot()
	kind, err := schedulemessage.KindOf(turn.Kind)
	if err != nil {
		return nil, fmt.Errorf("failed to read what plays: %w", err)
	}
	answer := &gamemasterv1.Turn{Kind: kind, Description: string(turn.Description)}
	if !turn.EndsAt.IsZero() {
		answer.EndsAt = timestamppb.New(turn.EndsAt)
	}
	battle, ok := plan.BattleOf(turn.Entry)
	if turn.Finale {
		battle, ok = plan.FinaleAt(now)
	}
	if !turn.EndsAt.IsZero() && !turn.Finale {
		answer.EntryId = turn.Entry.String()
	}
	if ok && !turn.EndsAt.IsZero() {
		answer.RoundId, answer.Times, answer.EndsSeason = string(battle.Round()), uint32(battle.Times), battle.EndsSeason
	}
	if since := sched.SinceAt(now); !since.IsZero() {
		answer.Since = timestamppb.New(since)
	}
	return answer, nil
}

// The battles to come by start, the finales among them; a tie goes to the entry made first, then the finale.
func timelineOf(battles []schedule.EntrySnapshot, finales []moments.Battle, plan moments.Plan) *gamemasterv1.Timeline {
	timeline := &gamemasterv1.Timeline{Battles: make([]*gamemasterv1.Battle, 0, len(battles)+len(finales))}
	for _, entry := range battles {
		battle := &gamemasterv1.Battle{
			EntryId: entry.ID.String(), StartsAt: timestamppb.New(entry.StartsAt), EndsAt: timestamppb.New(entry.EndsAt),
			Description: string(entry.Description), Times: uint32(entry.Times),
		}
		if planned, ok := plan.BattleOf(entry.ID); ok {
			battle.RoundId = string(planned.Round())
		}
		timeline.Battles = append(timeline.Battles, battle)
	}
	for _, finale := range finales {
		timeline.Battles = append(timeline.Battles, &gamemasterv1.Battle{
			StartsAt: timestamppb.New(finale.StartsAt), EndsAt: timestamppb.New(finale.EndsAt),
			Description: string(finale.Description), Times: uint32(finale.Times), EndsSeason: true,
			RoundId: string(finale.Round()),
		})
	}
	slices.SortStableFunc(timeline.Battles, func(a, b *gamemasterv1.Battle) int {
		return a.GetStartsAt().AsTime().Compare(b.GetStartsAt().AsTime())
	})
	return timeline
}

// The latest round or book scoring told, a book first at one time: it closes after its rounds.
func (q *PostgresQuery) debrief(ctx context.Context, plan moments.Plan) (*gamemasterv1.Debrief, error) {
	var (
		book    bool
		id      string
		endedAt time.Time
	)
	err := q.db.On(ctx).QueryRowContext(ctx, `
		SELECT book, id, ended_at FROM (
			SELECT false AS book, round_id AS id, ended_at FROM scored_rounds
			UNION ALL
			SELECT true, book_id, ended_at FROM closed_books
		) heard
		ORDER BY ended_at DESC, book DESC, id
		LIMIT 1`).Scan(&book, &id, &endedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no debrief before scoring scored any.
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the last moment scoring scored: %w", err)
	}

	var moment moments.Moment
	if book {
		moment, err = plan.BookClosed(moments.BookID(id), endedAt)
	} else {
		moment, err = plan.RoundEnded(moments.RoundID(id), endedAt)
	}
	if errors.Is(err, moments.ErrUnknownRound) || errors.Is(err, moments.ErrUnknownBook) {
		return nil, nil //nolint:nilnil // a moment the plan no longer knows is told no more.
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the last moment scoring scored: %w", err)
	}
	return momentmessage.DebriefOf(moment) //nolint:wrapcheck // it names the moment.
}

func (q *PostgresQuery) opened(ctx context.Context) ([]moments.RoundID, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT round_id FROM opened_rounds`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rounds opened: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var opened []moments.RoundID
	for rows.Next() {
		var round string
		if err := rows.Scan(&round); err != nil {
			return nil, fmt.Errorf("failed to read a round opened: %w", err)
		}
		opened = append(opened, moments.RoundID(round))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rounds opened: %w", err)
	}
	return opened, nil
}

func (q *PostgresQuery) entries(ctx context.Context) ([]schedule.Entry, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, rule_set, kind, description, starts_at, ends_at, priority, times, made_at FROM schedule_entries`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
		entry.Times = schedule.Times(times)                                                                //nolint:gosec // CHECK times BETWEEN 0 AND 10.
		entries = append(entries, entry.Restore())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	return entries, nil
}

func (q *PostgresQuery) seasons(ctx context.Context) (season.Calendar, error) {
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
