package levels

// XP is how far an account went, in one number. It is never stored: it is computed from the stats row at each read.
type XP uint64

// Record is what an account did, as its stats row keeps it. stats.Stats already answers the counts it keeps under the
// same names; the days played and the missions paid by kind come with the changes that add them to the stats.
type Record interface {
	TilesTaken() uint64
	DaysPlayed() uint64
	MissionGoals(kind MissionKind) uint64
	BombsDropped() uint64
	BonusesClaimed() uint64
	RightAnswers() uint64
	TilesFortified() uint64
}

// A source is one thing a player does that gives XP. A new source is a new type.
type Source interface {
	XP(record Record) XP
}

// A tile taken is the base unit.
type TilesTaken struct{}

func (TilesTaken) XP(record Record) XP { return XP(record.TilesTaken()) }

// A day played is the first take of a UTC day: a reason to come back each day.
type DaysPlayed struct{}

func (DaysPlayed) XP(record Record) XP { return XP(record.DaysPlayed() * 100) }

// A mission pays its goal times its kind's rate.
type MissionsDone struct{}

func (MissionsDone) Rates() []MissionRate {
	return []MissionRate{
		TakeFor{}, TakeFrom{}, TakeIn{}, TakeFromIn{}, CatchBoxes{}, AnswerQuizzes{}, ShieldTiles{}, FortifyTiles{}, BombIn{},
	}
}

func (m MissionsDone) XP(record Record) XP {
	var xp XP
	for _, rate := range m.Rates() {
		xp += rate.XP(record.MissionGoals(rate.Kind()))
	}
	return xp
}

// A bomb is a rare box.
type BombsDropped struct{}

func (BombsDropped) XP(record Record) XP { return XP(record.BombsDropped() * 50) }

// A box caught rewards watching the sky.
type BoxesCaught struct{}

func (BoxesCaught) XP(record Record) XP { return XP(record.BonusesClaimed() * 30) }

// A wrong answer gives nothing, as on the Quiz track.
type RightAnswers struct{}

func (RightAnswers) XP(record Record) XP { return XP(record.RightAnswers() * 30) }

// A tile fortified is 0.2 XP, floored on the total: a fortify counts the whole landmass, often thousands of tiles.
type TilesFortified struct{}

func (TilesFortified) XP(record Record) XP { return XP(record.TilesFortified() / 5) }

// A message sent gives nothing: XP for messages would fill the chat with spam.
type MessagesSent struct{}

func (MessagesSent) XP(Record) XP { return 0 }
