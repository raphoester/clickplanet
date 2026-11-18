package levels

// XP is how far an account went, in one number. It is never stored: it is computed from the stats row at each read.
type XP uint64

// Record is what an account did, as its stats row keeps it: stats.Stats is one.
type Record interface {
	TilesTaken() uint64
	DaysPlayed() uint64
	MissionGoals(kind MissionKind) uint64
	BombsDropped() uint64
	BonusesClaimed() uint64
	RightAnswers() uint64
	TilesFortified() uint64
}

// SourceName is a source's name: progression.v1.XpSource without its prefix, in lower case.
type SourceName string

// A source is one thing a player does that gives XP. A new source is a new type.
type Source interface {
	Name() SourceName
	XP(record Record) XP
}

// A rated source pays one rate on one count of the record, which the page shows. The missions pay by kind instead.
type RatedSource interface {
	Source
	Rate() Rate
}

// A tile taken is the base unit.
type TilesTaken struct{}

func (TilesTaken) Name() SourceName { return "tiles_taken" }

func (TilesTaken) Rate() Rate { return Rate{XP: 1, Per: 1} }

func (s TilesTaken) XP(record Record) XP { return s.Rate().XPOf(record.TilesTaken()) }

// A day played is the first take of a UTC day: a reason to come back each day.
type DaysPlayed struct{}

func (DaysPlayed) Name() SourceName { return "days_played" }

func (DaysPlayed) Rate() Rate { return Rate{XP: 100, Per: 1} }

func (s DaysPlayed) XP(record Record) XP { return s.Rate().XPOf(record.DaysPlayed()) }

// A mission pays its goal times its kind's rate.
type MissionsDone struct{}

func (MissionsDone) Name() SourceName { return "missions_done" }

func (MissionsDone) Rates() []MissionRate {
	return []MissionRate{
		TakeFor{}, TakeFrom{}, TakeIn{}, TakeFromIn{}, CatchBoxes{}, AnswerQuizzes{}, ShieldTiles{}, FortifyTiles{}, BombIn{},
	}
}

func (m MissionsDone) XP(record Record) XP {
	var xp XP
	for _, rate := range m.Rates() {
		xp += rate.Rate().XPOf(record.MissionGoals(rate.Kind()))
	}
	return xp
}

// A bomb is a rare box.
type BombsDropped struct{}

func (BombsDropped) Name() SourceName { return "bombs_dropped" }

func (BombsDropped) Rate() Rate { return Rate{XP: 50, Per: 1} }

func (s BombsDropped) XP(record Record) XP { return s.Rate().XPOf(record.BombsDropped()) }

// A box caught rewards watching the sky.
type BoxesCaught struct{}

func (BoxesCaught) Name() SourceName { return "boxes_caught" }

func (BoxesCaught) Rate() Rate { return Rate{XP: 30, Per: 1} }

func (s BoxesCaught) XP(record Record) XP { return s.Rate().XPOf(record.BonusesClaimed()) }

// A wrong answer gives nothing, as on the Quiz track.
type RightAnswers struct{}

func (RightAnswers) Name() SourceName { return "right_answers" }

func (RightAnswers) Rate() Rate { return Rate{XP: 30, Per: 1} }

func (s RightAnswers) XP(record Record) XP { return s.Rate().XPOf(record.RightAnswers()) }

// A tile fortified is 0.2 XP, floored on the total: a fortify counts the whole landmass, often thousands of tiles.
type TilesFortified struct{}

func (TilesFortified) Name() SourceName { return "tiles_fortified" }

func (TilesFortified) Rate() Rate { return Rate{XP: 1, Per: 5} }

func (s TilesFortified) XP(record Record) XP { return s.Rate().XPOf(record.TilesFortified()) }

// A message sent gives nothing: XP for messages would fill the chat with spam.
type MessagesSent struct{}

func (MessagesSent) Name() SourceName { return "messages_sent" }

func (MessagesSent) XP(Record) XP { return 0 }
