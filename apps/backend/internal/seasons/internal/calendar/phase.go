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

// PhaseAt is the current season and how far it is; past the last season, that season over. An empty calendar has none.
func (c Calendar) PhaseAt(now time.Time) (Phase, bool) {
	if len(c.seasons) == 0 {
		return Phase{}, false
	}

	season, ok := c.Current(now)
	switch {
	case !ok:
		return Phase{Season: c.seasons[len(c.seasons)-1].Number, Stage: StageOver}, true
	case now.Before(season.FinaleStartsAt):
		return Phase{Season: season.Number, Stage: StageDays}, true
	default:
		return Phase{Season: season.Number, Stage: StageFinale}, true
	}
}
