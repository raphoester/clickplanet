package calendar

import "time"

type Stage string

const (
	StageDays   Stage = "days"
	StageFinale Stage = "finale"
	StageOver   Stage = "over"
)

// Phase is a stretch of a season: its days, its finale, or the time after the last season.
type Phase struct {
	Season   Number
	Stage    Stage
	StartsAt time.Time
	EndsAt   time.Time
}

// Phases are every season's days then finale, and the last season over, which never ends (a zero EndsAt). The first
// season's days started before anything (a zero StartsAt). An empty calendar has none.
func (c Calendar) Phases() []Phase {
	if len(c.seasons) == 0 {
		return nil
	}

	phases := make([]Phase, 0, 2*len(c.seasons)+1)
	var started time.Time
	for _, season := range c.seasons {
		phases = append(phases,
			Phase{Season: season.Number, Stage: StageDays, StartsAt: started, EndsAt: season.FinaleStartsAt},
			Phase{Season: season.Number, Stage: StageFinale, StartsAt: season.FinaleStartsAt, EndsAt: season.EndsAt},
		)
		started = season.EndsAt
	}
	last := c.seasons[len(c.seasons)-1]
	return append(phases, Phase{Season: last.Number, Stage: StageOver, StartsAt: last.EndsAt})
}
