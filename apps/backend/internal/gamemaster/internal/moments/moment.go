package moments

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// Moment is what the gamemaster tells, in the game's words: a day started, or a day, a battle or a season scoring
// scored.
type Moment interface {
	moment()
}

type DayStarted struct {
	Day season.Day
}

type DayEnded struct {
	Day     season.Day
	EndedAt time.Time
}

type BattleEnded struct {
	Battle  Battle
	EndedAt time.Time
}

type SeasonEnded struct {
	Season season.Number
	Book   BookID
	Finale RoundID
	// FinaleBattle is the finale when it is a battle; nil otherwise.
	FinaleBattle *BattleEnded
	EndedAt      time.Time
}

func (DayStarted) moment()  {}
func (DayEnded) moment()    {}
func (BattleEnded) moment() {}
func (SeasonEnded) moment() {}
