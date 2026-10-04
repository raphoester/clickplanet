package playerread

import (
	"errors"
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

var ErrUnknownColor = errors.New("a kept color the proto does not name")

func KeptColor(value int32) (playerv1.NameColor, error) {
	if _, named := playerv1.NameColor_name[value]; !named {
		return 0, fmt.Errorf("%w: %d", ErrUnknownColor, value)
	}
	return playerv1.NameColor(value), nil
}

// Needs the stats row joined as stats and today's UTC date as $2: Streak.AsOf for the commands.
const StreakNow = `CASE WHEN stats.streak_last_day IS NULL OR stats.streak_last_day IN ($2::date, $2::date - 1)
	THEN COALESCE(stats.streak_current, 0) ELSE 0 END`

type Career struct {
	TilesTaken   uint64
	StreakNow    uint32
	MessagesSent uint64
}
