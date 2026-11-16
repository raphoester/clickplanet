package levels

import "github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"

type MissionKind = stats.MissionKind

// A mission rate is the XP a mission of its kind pays per unit of its goal, so a bigger goal pays more. Missions tells
// the goals; progression owns the rates. A new kind of mission is a new type.
type MissionRate interface {
	Kind() MissionKind
	Rate() Rate
}

// Rate is exact: XP for every Per units of the goal, floored, so the page that reads it pays what the server does.
type Rate struct {
	XP  XP
	Per uint64
}

func (r Rate) XPOf(goals uint64) XP { return XP(goals * uint64(r.XP) / r.Per) }

// Two to four banks.
type TakeFor struct{}

func (TakeFor) Kind() MissionKind { return "take_for" }

func (TakeFor) Rate() Rate { return Rate{XP: 1, Per: 1} }

// A rival's land is fought over and often shielded: 1.5 per tile, floored.
type TakeFrom struct{}

func (TakeFrom) Kind() MissionKind { return "take_from" }

func (TakeFrom) Rate() Rate { return Rate{XP: 3, Per: 2} }

// The longest: 12 to 25 minutes.
type TakeIn struct{}

func (TakeIn) Kind() MissionKind { return "take_in" }

func (TakeIn) Rate() Rate { return Rate{XP: 1, Per: 1} }

// One place, defended.
type TakeFromIn struct{}

func (TakeFromIn) Kind() MissionKind { return "take_from_in" }

func (TakeFromIn) Rate() Rate { return Rate{XP: 3, Per: 1} }

// A box comes every 4 to 8 minutes.
type CatchBoxes struct{}

func (CatchBoxes) Kind() MissionKind { return "catch_boxes" }

func (CatchBoxes) Rate() Rate { return Rate{XP: 50, Per: 1} }

// 44% of answers are right: 2 right takes about 5 quizzes.
type AnswerQuizzes struct{}

func (AnswerQuizzes) Kind() MissionKind { return "answer_quizzes" }

func (AnswerQuizzes) Rate() Rate { return Rate{XP: 80, Per: 1} }

// It needs shield boxes.
type ShieldTiles struct{}

func (ShieldTiles) Kind() MissionKind { return "shield_tiles" }

func (ShieldTiles) Rate() Rate { return Rate{XP: 80, Per: 1} }

// It needs a whole landmass.
type FortifyTiles struct{}

func (FortifyTiles) Kind() MissionKind { return "fortify_tiles" }

func (FortifyTiles) Rate() Rate { return Rate{XP: 4, Per: 1} }

// A bomb is 1 box in 14.
type BombIn struct{}

func (BombIn) Kind() MissionKind { return "bomb_in" }

func (BombIn) Rate() Rate { return Rate{XP: 150, Per: 1} }
