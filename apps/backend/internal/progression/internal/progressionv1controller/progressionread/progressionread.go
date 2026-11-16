package progressionread

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
	MissionGoals   map[string]uint64
	BombsDropped   uint64
	BonusesClaimed uint64
	RightAnswers   uint64
	TilesFortified uint64
}
