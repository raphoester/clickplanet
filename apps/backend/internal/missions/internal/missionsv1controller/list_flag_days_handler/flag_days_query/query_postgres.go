package flag_days_query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const longest = 7 * 24 * time.Hour

var (
	ErrInvalidWindow = errors.New("the flag days are read over a since before its until, at most 7 days apart")
	ErrUnknownKind   = errors.New("not a kind of flag mission")
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// A day nobody played still has its row, with no flag.
const flagDays = `
	SELECT d.day_id, d.starts_at, d.ends_at, f.flag, f.takes, f.strikes,
	       m.mission_id, m.kind::text, m.payload, m.points, m.progress, m.done
	FROM mission_days AS d
	LEFT JOIN flag_days AS f ON f.day = d.day_id
	LEFT JOIN flag_missions AS m ON m.day = f.day AND m.flag = f.flag
	WHERE d.starts_at >= $1 AND d.starts_at < $2
	ORDER BY d.starts_at, f.flag`

// Unset until is now.
func (q *PostgresQuery) FlagDays(ctx context.Context, since, until time.Time) (*missionsv1.ListFlagDaysResponse, error) {
	if until.IsZero() {
		until = q.clock.Now()
	}
	if since.IsZero() || !since.Before(until) || until.Sub(since) > longest {
		return nil, ErrInvalidWindow
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, flagDays, since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flag days: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var (
		answer  = &missionsv1.ListFlagDaysResponse{}
		current *missionsv1.FlagDays
	)
	for rows.Next() {
		var (
			dayID            string
			startsAt, endsAt time.Time
			flag             sql.NullString
			takes, strikes   sql.NullInt64
			mission          uuid.NullUUID
			kind             sql.NullString
			payload          []byte
			points, progress sql.NullInt64
			done             sql.NullBool
		)
		if err := rows.Scan(&dayID, &startsAt, &endsAt, &flag, &takes, &strikes, &mission, &kind, &payload, &points,
			&progress, &done); err != nil {
			return nil, fmt.Errorf("failed to read a flag day: %w", err)
		}

		if current.GetDay().GetId() != dayID {
			current = &missionsv1.FlagDays{Day: &missionsv1.Day{
				Id: dayID, StartsAt: timestamppb.New(startsAt), EndsAt: timestamppb.New(endsAt),
			}}
			answer.Days = append(answer.Days, current)
		}
		if !flag.Valid {
			continue
		}

		//nolint:gosec // CHECK (... >= 0) on each.
		day := &missionsv1.FlagDay{CountryId: flag.String, TilesTaken: uint64(takes.Int64), ShieldsBroken: uint64(strikes.Int64)}
		if mission.Valid {
			//nolint:gosec // CHECK (points > 0), CHECK (progress >= 0).
			day.Mission, err = missionOf(mission.UUID, kind.String, payload, uint32(points.Int64), uint32(progress.Int64), done.Bool)
			if err != nil {
				return nil, err
			}
		}
		current.Flags = append(current.Flags, day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the flag days: %w", err)
	}
	return answer, nil
}

// The payload is postgres_flag_day_store's: the parity test reads back every kind it keeps.
func missionOf(id uuid.UUID, kind string, payload []byte, points, progress uint32, done bool) (*missionsv1.FlagMission, error) {
	mission := &missionsv1.FlagMission{Id: id.String(), Points: points, Done: done}
	switch kind {
	case "take_in":
		var record struct {
			Ground string `json:"ground"`
			Goal   uint32 `json:"goal"`
		}
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("failed to read a conquest: %w", err)
		}
		mission.Kind = &missionsv1.FlagMission_TakeIn{TakeIn: &missionsv1.TakeIn{
			CountryId: record.Ground, Goal: record.Goal, Taken: progress,
		}}
	case "break_shields":
		var record struct {
			Flag string `json:"flag"`
			Goal uint32 `json:"goal"`
		}
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("failed to read a shield mission: %w", err)
		}
		mission.Kind = &missionsv1.FlagMission_BreakShields{BreakShields: &missionsv1.BreakShields{
			CountryId: record.Flag, Goal: record.Goal, Broken: progress,
		}}
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	return mission, nil
}
