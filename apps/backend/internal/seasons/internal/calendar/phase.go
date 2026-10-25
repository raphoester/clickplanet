package calendar

import "time"

type Stage string

const (
	StageDays   Stage = "days"
	StageFinale Stage = "finale"
	StageOver   Stage = "over"
)

type Phase struct {
	Season Number
	Stage  Stage
}

// PhaseAt is the current season and how far it is; between two seasons and past the last, the season that ended over.
// Before the first season starts, and in an empty calendar, there is none.
func (c Calendar) PhaseAt(now time.Time) (Phase, bool) {
	if season, ok := c.Current(now); ok {
		if now.Before(season.FinaleStartsAt) {
			return Phase{Season: season.Number, Stage: StageDays}, true
		}
		return Phase{Season: season.Number, Stage: StageFinale}, true
	}

	ended, ok := c.ended(now)
	if !ok {
		return Phase{}, false
	}
	return Phase{Season: ended.Number, Stage: StageOver}, true
}
