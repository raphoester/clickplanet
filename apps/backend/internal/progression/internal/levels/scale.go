package levels

import "github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"

// Scale weighs what an account did into XP and places it on the curve.
type Scale struct{}

func (Scale) Sources() []Source {
	return []Source{
		TilesTaken{},
		DaysPlayed{},
		MissionsDone{},
		BombsDropped{},
		BoxesCaught{},
		RightAnswers{},
		TilesFortified{},
		MessagesSent{},
	}
}

func (s Scale) XPOf(record Record) XP {
	var xp XP
	for _, source := range s.Sources() {
		xp += source.XP(record)
	}
	return xp
}

func (s Scale) Reach(record Record) Level {
	return Curve{}.LevelOf(s.XPOf(record))
}

// StandingOf is where the account's record puts it. A guest has no XP and no level, whatever it did.
func (s Scale) StandingOf(account accounts.Account, record Record) Standing {
	if !account.Linked() {
		return Standing{}
	}
	return Curve{}.StandingOf(s.XPOf(record))
}

func (s Scale) StandingAt(kept Level, record Record) Standing {
	if kept == 0 {
		return Standing{}
	}
	return Curve{}.StandingAt(kept, s.XPOf(record))
}
