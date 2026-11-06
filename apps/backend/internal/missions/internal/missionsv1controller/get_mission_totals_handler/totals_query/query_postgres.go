package totals_query

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const longest = 31 * 24 * time.Hour

var (
	ErrInvalidWindow = errors.New("the totals are read over a since before its until, at most 31 days apart")
	ErrUnknownKind   = errors.New("not a kind of mission")
)

// The names postgres_mission_store keeps each kind under: the parity test reads back every kind it keeps.
var kinds = map[string]missionsv1.MissionKind{
	"take_for":       missionsv1.MissionKind_MISSION_KIND_TAKE_FOR,
	"take_from":      missionsv1.MissionKind_MISSION_KIND_TAKE_FROM,
	"take_in":        missionsv1.MissionKind_MISSION_KIND_TAKE_IN,
	"take_from_in":   missionsv1.MissionKind_MISSION_KIND_TAKE_FROM_IN,
	"catch_boxes":    missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES,
	"answer_quizzes": missionsv1.MissionKind_MISSION_KIND_ANSWER_QUIZZES,
	"bomb_in":        missionsv1.MissionKind_MISSION_KIND_BOMB_IN,
	"shield_tiles":   missionsv1.MissionKind_MISSION_KIND_SHIELD_TILES,
	"fortify_tiles":  missionsv1.MissionKind_MISSION_KIND_FORTIFY_TILES,
}

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// Every mission of the briefs of the days told in the window. Every kind's order is done once its progress reaches its
// goal, and progress stops there.
const drawn = `
	WITH told AS (
		SELECT day_id, starts_at, ends_at FROM mission_days WHERE starts_at >= $1 AND starts_at < $2
	), drawn AS (
		SELECT b.day, b.account_id, m->>'kind' AS kind, (m->'payload'->>'goal')::bigint AS goal,
		       COALESCE(p.progress, 0)::bigint AS progress, pay.mission_id IS NOT NULL AS paid
		FROM told
		JOIN mission_briefs AS b ON b.day = told.day_id
		CROSS JOIN LATERAL jsonb_array_elements(b.missions) AS m
		LEFT JOIN mission_progress AS p
			ON p.day = b.day AND p.account_id = b.account_id AND p.mission_id = (m->>'id')::uuid
		LEFT JOIN mission_payments AS pay
			ON pay.day = b.day AND pay.account_id = b.account_id AND pay.mission_id = (m->>'id')::uuid
	)`

const byDay = drawn + `
	SELECT told.day_id, told.starts_at, told.ends_at, count(a.account_id),
	       count(a.account_id) FILTER (WHERE a.done > 0), count(a.account_id) FILTER (WHERE a.done = a.drawn)
	FROM told LEFT JOIN (
		SELECT day, account_id, count(*) AS drawn, count(*) FILTER (WHERE progress >= goal) AS done
		FROM drawn GROUP BY day, account_id
	) AS a ON a.day = told.day_id
	GROUP BY told.day_id, told.starts_at, told.ends_at
	ORDER BY told.starts_at`

const byKind = drawn + `
	SELECT day, kind, count(*), count(*) FILTER (WHERE progress > 0), count(*) FILTER (WHERE progress >= goal),
	       count(*) FILTER (WHERE paid), sum(goal), sum(progress)
	FROM drawn GROUP BY day, kind`

// Unset until is now.
func (q *PostgresQuery) Totals(ctx context.Context, since, until time.Time) (*missionsv1.GetMissionTotalsResponse, error) {
	if until.IsZero() {
		until = q.clock.Now()
	}
	if since.IsZero() || !since.Before(until) || until.Sub(since) > longest {
		return nil, ErrInvalidWindow
	}

	days, err := q.days(ctx, since, until)
	if err != nil {
		return nil, err
	}
	if err := q.kinds(ctx, since, until, days); err != nil {
		return nil, err
	}
	for _, day := range days {
		slices.SortFunc(day.Kinds, func(a, b *missionsv1.KindTotals) int { return cmp.Compare(a.GetKind(), b.GetKind()) })
	}
	return &missionsv1.GetMissionTotalsResponse{Days: days}, nil
}

func (q *PostgresQuery) days(ctx context.Context, since, until time.Time) ([]*missionsv1.DayTotals, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, byDay, since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the days' briefs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var days []*missionsv1.DayTotals
	for rows.Next() {
		var (
			day              = &missionsv1.DayTotals{Day: &missionsv1.Day{}}
			startsAt, endsAt time.Time
		)
		if err := rows.Scan(&day.Day.Id, &startsAt, &endsAt, &day.Briefs, &day.FinishedAny, &day.FinishedAll); err != nil {
			return nil, fmt.Errorf("failed to read a day's briefs: %w", err)
		}
		day.Day.StartsAt, day.Day.EndsAt = timestamppb.New(startsAt), timestamppb.New(endsAt)
		days = append(days, day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the days' briefs: %w", err)
	}
	return days, nil
}

func (q *PostgresQuery) kinds(ctx context.Context, since, until time.Time, days []*missionsv1.DayTotals) error {
	rows, err := q.db.On(ctx).QueryContext(ctx, byKind, since, until)
	if err != nil {
		return fmt.Errorf("failed to read the days' missions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byID := make(map[string]*missionsv1.DayTotals, len(days))
	for _, day := range days {
		byID[day.GetDay().GetId()] = day
	}
	for rows.Next() {
		var (
			dayID, name string
			totals      = &missionsv1.KindTotals{}
		)
		if err := rows.Scan(&dayID, &name, &totals.Drawn, &totals.Started, &totals.Done, &totals.Paid, &totals.Goals,
			&totals.Progress); err != nil {
			return fmt.Errorf("failed to read a day's missions: %w", err)
		}
		kind, ok := kinds[name]
		if !ok {
			return fmt.Errorf("%w: %q", ErrUnknownKind, name)
		}
		totals.Kind = kind
		// A day told between the two reads is not in the answer.
		if day, ok := byID[dayID]; ok {
			day.Kinds = append(day.Kinds, totals)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the days' missions: %w", err)
	}
	return nil
}
