package admin_progress_query

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Titles interface {
	Shown(held []string) []*progressionv1.Title
	Worn(held []string, choice string) *progressionv1.Title
}

type XP interface {
	XP(record progressionread.Record) *progressionv1.XpBreakdown
}

type Fronts interface {
	Fronts(ctx context.Context, account accounts.AccountID) (*progressionv1.GetFrontsResponse, error)
}

func NewPostgresQuery(db cppg.Conn, titles Titles, xp XP, fronts Fronts, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, titles: titles, xp: xp, fronts: fronts, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Conn
	titles Titles
	xp     XP
	fronts Fronts
	clock  cptime.Clock
}

// The stats as kept, a guest's streak included: what the operator reads is not what the game shows.
const progress = `
	SELECT
		COALESCE(stats.tiles_taken, 0),
		` + progressionread.StreakNow + `,
		COALESCE(stats.streak_best, 0),
		COALESCE(to_char(stats.streak_last_day, 'YYYY-MM-DD'), ''),
		COALESCE(stats.messages_sent, 0),
		COALESCE(stats.days_played, 0),
		COALESCE(stats.bombs_dropped, 0),
		COALESCE(stats.bonuses_claimed, 0),
		COALESCE(stats.right_answers, 0),
		COALESCE(stats.tiles_fortified, 0),
		COALESCE(stats.missions_by_kind, '{}'),
		COALESCE(levels.level, 0),
		COALESCE(worn_titles.title, ''),
		COALESCE(
			(SELECT array_agg(titles.title ORDER BY titles.earned_at, titles.title) FROM titles WHERE titles.account_id = asked.account_id),
			'{}'
		)
	FROM (SELECT $1::uuid AS account_id) asked
	LEFT JOIN stats ON stats.account_id = asked.account_id
	LEFT JOIN levels ON levels.account_id = asked.account_id
	LEFT JOIN worn_titles ON worn_titles.account_id = asked.account_id
`

func (q *PostgresQuery) Progress(ctx context.Context, account accounts.AccountID) (*progressionv1.AdminGetProgressResponse, error) {
	var (
		choice   string
		held     []string
		missions []byte
		record   progressionread.Record
		answer   = &progressionv1.AdminGetProgressResponse{Stats: &progressionv1.Stats{}}
	)
	if err := q.db.On(ctx).QueryRowContext(ctx, progress, uuid.UUID(account), q.clock.Now().UTC().Format(time.DateOnly)).Scan(
		&answer.Stats.TilesTaken, &answer.Stats.StreakCurrent, &answer.Stats.StreakBest, &answer.Stats.StreakLastDay,
		&answer.MessagesSent, &answer.DaysPlayed, &record.BombsDropped, &record.BonusesClaimed, &record.RightAnswers,
		&record.TilesFortified, &missions, &answer.Level, &choice, pq.Array(&held),
	); err != nil {
		return nil, fmt.Errorf("failed to read the progress: %w", err)
	}
	var tallies map[string]tally
	if err := json.Unmarshal(missions, &tallies); err != nil {
		return nil, fmt.Errorf("failed to read the missions by kind: %w", err)
	}
	record.TilesTaken, record.DaysPlayed = answer.GetStats().GetTilesTaken(), answer.GetDaysPlayed()
	record.MissionGoals = make(map[string]uint64, len(tallies))
	for _, kind := range slices.Sorted(maps.Keys(tallies)) {
		record.MissionGoals[kind] = tallies[kind].Goals
		answer.MissionsByKind = append(answer.MissionsByKind,
			&progressionv1.MissionTally{Kind: kind, Done: tallies[kind].Done, Goals: tallies[kind].Goals})
	}
	answer.Xp = q.xp.XP(record)

	fronts, err := q.fronts.Fronts(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to ask which countries the player plays for and against: %w", err)
	}
	answer.PlaysFor, answer.PlaysAgainst = fronts.GetPlaysFor(), fronts.GetPlaysAgainst()
	answer.Titles = q.titles.Shown(held)
	answer.WornTitle = q.titles.Worn(held, choice)
	return answer, nil
}

type tally struct {
	Done  uint64 `json:"done"`
	Goals uint64 `json:"goals"`
}
