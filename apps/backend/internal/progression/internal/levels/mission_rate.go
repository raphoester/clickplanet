package levels

import "github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"

type MissionKind = stats.MissionKind

// A mission rate is the XP a mission of its kind pays per unit of its goal, so a bigger goal pays more. Missions tells
// the goals; progression owns the rates. A new kind of mission is a new type.
type MissionRate interface {
	Kind() MissionKind
	XP(goals uint64) XP
}

// Two to four banks.
type TakeFor struct{}

func (TakeFor) Kind() MissionKind { return "take_for" }

func (TakeFor) XP(goals uint64) XP { return XP(goals) }

// A rival's land is fought over and often shielded: 1.5 per tile, floored.
type TakeFrom struct{}

func (TakeFrom) Kind() MissionKind { return "take_from" }

func (TakeFrom) XP(goals uint64) XP { return XP(goals * 3 / 2) }

// The longest: 12 to 25 minutes.
type TakeIn struct{}

func (TakeIn) Kind() MissionKind { return "take_in" }

func (TakeIn) XP(goals uint64) XP { return XP(goals) }

// One place, defended.
type TakeFromIn struct{}

func (TakeFromIn) Kind() MissionKind { return "take_from_in" }

func (TakeFromIn) XP(goals uint64) XP { return XP(goals * 3) }

// A box comes every 4 to 8 minutes.
type CatchBoxes struct{}

func (CatchBoxes) Kind() MissionKind { return "catch_boxes" }

func (CatchBoxes) XP(goals uint64) XP { return XP(goals * 50) }

// 44% of answers are right: 2 right takes about 5 quizzes.
type AnswerQuizzes struct{}

func (AnswerQuizzes) Kind() MissionKind { return "answer_quizzes" }

func (AnswerQuizzes) XP(goals uint64) XP { return XP(goals * 80) }

// It needs shield boxes.
type ShieldTiles struct{}

func (ShieldTiles) Kind() MissionKind { return "shield_tiles" }

func (ShieldTiles) XP(goals uint64) XP { return XP(goals * 80) }

// It needs a whole landmass.
type FortifyTiles struct{}

func (FortifyTiles) Kind() MissionKind { return "fortify_tiles" }

func (FortifyTiles) XP(goals uint64) XP { return XP(goals * 4) }

// A bomb is 1 box in 14.
type BombIn struct{}

func (BombIn) Kind() MissionKind { return "bomb_in" }

func (BombIn) XP(goals uint64) XP { return XP(goals * 150) }
