package progressionread

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Needs the stats row joined as stats and today's UTC date as $2: Streak.AsOf for the commands.
const StreakNow = `CASE WHEN stats.streak_last_day IS NULL OR stats.streak_last_day IN ($2::date, $2::date - 1)
	THEN COALESCE(stats.streak_current, 0) ELSE 0 END`

type Career struct {
	TilesTaken     uint64
	StreakNow      uint32
	MessagesSent   uint64
	BombsDropped   uint64
	TilesFortified uint64
	BonusesClaimed uint64
	RightAnswers   uint64
	MissionsDone   uint64
}

// Record is what the level's XP is read from, as the stats row keeps it: each kind of mission's goals by its name.
type Record struct {
	TilesTaken     uint64
	DaysPlayed     uint64
	MissionGoals   MissionGoals
	BombsDropped   uint64
	BonusesClaimed uint64
	RightAnswers   uint64
	TilesFortified uint64
}

// Needs the stats row joined as stats, perhaps missing: a Record, in the order of Record.Into.
const RecordColumns = `COALESCE(stats.tiles_taken, 0), COALESCE(stats.days_played, 0),
	COALESCE((SELECT jsonb_object_agg(kinds.key, kinds.value->'goals') FROM jsonb_each(stats.missions_by_kind) kinds), '{}'),
	COALESCE(stats.bombs_dropped, 0), COALESCE(stats.bonuses_claimed, 0), COALESCE(stats.right_answers, 0),
	COALESCE(stats.tiles_fortified, 0)`

// Into is where a scan of RecordColumns goes.
func (r *Record) Into() []any {
	return []any{
		&r.TilesTaken, &r.DaysPlayed, &r.MissionGoals, &r.BombsDropped, &r.BonusesClaimed, &r.RightAnswers, &r.TilesFortified,
	}
}

// MissionGoals is each kind's goals, scanned from a JSON object of them.
type MissionGoals map[string]uint64

var ErrNotGoals = errors.New("the goals by kind are not a JSON object")

func (g *MissionGoals) Scan(src any) error {
	raw, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("%w: %T", ErrNotGoals, src)
	}
	if err := json.Unmarshal(raw, g); err != nil {
		return fmt.Errorf("%w: %w", ErrNotGoals, err)
	}
	return nil
}
